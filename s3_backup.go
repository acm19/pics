package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// RestoreFilter defines the date range filter for restoring backups
type RestoreFilter struct {
	FromYear  int // 0 means no lower bound
	FromMonth int // 0 means January if FromYear is set
	ToYear    int // 0 means no upper bound
	ToMonth   int // 0 means December if ToYear is set
}

// S3Backup defines the interface for backing up and restoring directories to/from S3
type S3Backup interface {
	// BackupDirectories backs up all subdirectories in the source directory to S3
	BackupDirectories(sourceDir, bucket string, maxConcurrent int) error
	// RestoreDirectories restores directories from S3 to target directory
	RestoreDirectories(bucket, targetDir string, filter RestoreFilter, maxConcurrent int) error
}

// s3Backup implements the S3Backup interface
type s3Backup struct {
	client     *s3.Client
	extensions Extensions
}

// NewS3Backup creates a new S3Backup instance
func NewS3Backup(ctx context.Context) (S3Backup, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &s3Backup{
		client:     s3.NewFromConfig(cfg),
		extensions: NewExtensions(),
	}, nil
}

// BackupDirectories backs up all subdirectories to S3 in parallel
func (b *s3Backup) BackupDirectories(sourceDir, bucket string, maxConcurrent int) error {
	// Find all subdirectories
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}

	if len(directories) == 0 {
		logger.Info("No directories found to backup")
		return nil
	}

	logger.Info("Starting S3 backup", "directories", len(directories), "bucket", bucket, "concurrency", maxConcurrent)

	// Create worker pool
	jobs := make(chan string, len(directories))
	results := make(chan error, len(directories))
	var wg sync.WaitGroup

	// Start workers
	for i := range maxConcurrent {
		wg.Add(1)
		go b.backupWorker(i, sourceDir, bucket, jobs, results, &wg)
	}

	// Send jobs
	for _, dirName := range directories {
		jobs <- dirName
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect errors
	var errors []error
	successCount := 0
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		logger.Error("Backup completed with errors", "successful", successCount, "failed", len(errors))
		return fmt.Errorf("backup failed for %d directories", len(errors))
	}

	logger.Info("Backup completed successfully", "directories_backed_up", successCount)
	return nil
}

// backupWorker processes backup jobs from the jobs channel
func (b *s3Backup) backupWorker(workerID int, sourceDir, bucket string, jobs <-chan string, results chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	for dirName := range jobs {
		logger.Debug("Worker processing directory", "worker", workerID, "directory", dirName)
		if err := b.backupDirectory(sourceDir, dirName, bucket); err != nil {
			logger.Error("Failed to backup directory", "directory", dirName, "error", err)
			results <- fmt.Errorf("directory %s: %w", dirName, err)
		} else {
			results <- nil
		}
	}
}

// countMediaFiles counts images and videos in a directory
func (b *s3Backup) countMediaFiles(dirPath string) (images int, videos int, err error) {
	// Count images
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, 0, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(dirPath, entry.Name())
		if b.extensions.IsImage(filePath) {
			images++
		}
	}

	// Count videos in videos subdirectory
	videosDir := filepath.Join(dirPath, "videos")
	if info, err := os.Stat(videosDir); err == nil && info.IsDir() {
		videoEntries, err := os.ReadDir(videosDir)
		if err != nil {
			return 0, 0, err
		}

		for _, entry := range videoEntries {
			if entry.IsDir() {
				continue
			}
			filePath := filepath.Join(videosDir, entry.Name())
			if b.extensions.IsVideo(filePath) {
				videos++
			}
		}
	}

	return images, videos, nil
}

// backupDirectory backs up a single directory to S3
func (b *s3Backup) backupDirectory(sourceDir, dirName, bucket string) error {
	dirPath := filepath.Join(sourceDir, dirName)

	// Count media files
	imageCount, videoCount, err := b.countMediaFiles(dirPath)
	if err != nil {
		return fmt.Errorf("failed to count media files: %w", err)
	}

	// Build S3 key with counts
	s3Key := fmt.Sprintf("%s (%d images, %d videos).tar.gz", dirName, imageCount, videoCount)

	// Create temporary tar.gz file
	tmpDir := fmt.Sprintf("/tmp/pics_tmp_%d", rand.Int())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		logger.Debug("Cleaning up temporary directory", "path", tmpDir)
		if err := os.RemoveAll(tmpDir); err != nil {
			logger.Error("Failed to remove temporary directory", "path", tmpDir, "error", err)
		}
	}()

	archivePath := filepath.Join(tmpDir, filepath.Base(s3Key))
	logger.Info("Creating archive", "directory", dirName, "images", imageCount, "videos", videoCount)

	if err := b.createTarGz(dirPath, archivePath); err != nil {
		return fmt.Errorf("failed to create tar.gz: %w", err)
	}

	// Calculate MD5 hash of the archive
	localHash, err := b.calculateMD5(archivePath)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5: %w", err)
	}

	// Check if object already exists in S3 with same hash
	ctx := context.Background()
	headOutput, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	})

	if err == nil {
		// Object exists, check if hash matches
		remoteETag := ""
		if headOutput.ETag != nil {
			remoteETag = *headOutput.ETag
			// Remove quotes from ETag
			remoteETag = remoteETag[1 : len(remoteETag)-1]
		}

		if remoteETag == localHash {
			logger.Info("Object already exists in S3 with matching hash, skipping", "directory", dirName, "key", s3Key, "hash", localHash)
			return nil
		}

		// Hash mismatch - fail with clear error
		return fmt.Errorf("hash mismatch for '%s': S3 object exists with different content (local: %s, remote: %s). Manual intervention required", s3Key, localHash, remoteETag)
	} else if !isNotFoundError(err) {
		return fmt.Errorf("failed to check S3 object existence: %w", err)
	}

	// Upload to S3
	logger.Info("Uploading to S3", "directory", dirName, "bucket", bucket, "key", s3Key, "hash", localHash)
	if err := b.uploadToS3(ctx, archivePath, bucket, s3Key); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	logger.Info("Successfully backed up directory", "directory", dirName, "key", s3Key)
	return nil
}

// calculateMD5 calculates the MD5 hash of a file
func (b *s3Backup) calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// isNotFoundError checks if the error is a NotFound error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for NotFound type directly
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}

	// Check for smithy API error with 404 status code
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorCode() == "NotFound" {
			return true
		}
	}

	// Check error message as fallback
	errMsg := err.Error()
	if strings.Contains(errMsg, "NotFound") || strings.Contains(errMsg, "StatusCode: 404") {
		return true
	}

	return false
}

// createTarGz creates a tar.gz archive of a directory
func (b *s3Backup) createTarGz(sourceDir, targetFile string) error {
	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Get the base directory name to include in archive paths
	baseName := filepath.Base(sourceDir)

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Update header name to include base directory name
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Include the directory name in the archive path
		if relPath == "." {
			header.Name = baseName
		} else {
			header.Name = filepath.Join(baseName, relPath)
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If not a regular file, skip writing content
		if !info.Mode().IsRegular() {
			return nil
		}

		// Write file content
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}

		return nil
	})
}

// uploadToS3 uploads a file to S3
func (b *s3Backup) uploadToS3(ctx context.Context, filePath, bucket, key string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})

	return err
}

// RestoreDirectories restores directories from S3 to target directory
func (b *s3Backup) RestoreDirectories(bucket, targetDir string, filter RestoreFilter, maxConcurrent int) error {
	ctx := context.Background()

	// List all objects in bucket
	logger.Info("Listing objects in S3 bucket", "bucket", bucket)
	var allObjects []types.Object
	paginator := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		allObjects = append(allObjects, page.Contents...)
	}

	// Filter objects based on date range
	var objectsToRestore []types.Object
	for _, obj := range allObjects {
		if obj.Key == nil {
			continue
		}
		if b.matchesFilter(*obj.Key, filter) {
			objectsToRestore = append(objectsToRestore, obj)
		}
	}

	if len(objectsToRestore) == 0 {
		logger.Info("No objects found matching filter")
		return nil
	}

	logger.Info("Starting restore", "objects", len(objectsToRestore), "target", targetDir, "concurrency", maxConcurrent)

	// Create worker pool
	jobs := make(chan types.Object, len(objectsToRestore))
	results := make(chan error, len(objectsToRestore))
	var wg sync.WaitGroup

	// Start workers
	for i := range maxConcurrent {
		wg.Add(1)
		go b.restoreWorker(i, bucket, targetDir, jobs, results, &wg)
	}

	// Send jobs
	for _, obj := range objectsToRestore {
		jobs <- obj
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect errors
	var errors []error
	successCount := 0
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		logger.Error("Restore completed with errors", "successful", successCount, "failed", len(errors))
		return fmt.Errorf("restore failed for %d objects", len(errors))
	}

	logger.Info("Restore completed successfully", "directories_restored", successCount)
	return nil
}

// restoreWorker processes restore jobs from the jobs channel
func (b *s3Backup) restoreWorker(workerID int, bucket, targetDir string, jobs <-chan types.Object, results chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()
	for obj := range jobs {
		logger.Debug("Worker processing object", "worker", workerID, "key", *obj.Key)
		if err := b.restoreObject(bucket, targetDir, *obj.Key); err != nil {
			logger.Error("Failed to restore object", "key", *obj.Key, "error", err)
			results <- fmt.Errorf("object %s: %w", *obj.Key, err)
		} else {
			results <- nil
		}
	}
}

// restoreObject downloads and extracts a single object from S3
func (b *s3Backup) restoreObject(bucket, targetDir, key string) error {
	ctx := context.Background()

	// Extract directory name from key (remove " (X images, Y videos).tar.gz" suffix)
	dirName := b.extractDirNameFromKey(key)
	targetPath := filepath.Join(targetDir, dirName)

	// Check if directory already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("directory already exists: %s", targetPath)
	}

	// Create temporary directory for download
	tmpDir := fmt.Sprintf("/tmp/pics_restore_%d", rand.Int())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		logger.Debug("Cleaning up temporary directory", "path", tmpDir)
		if err := os.RemoveAll(tmpDir); err != nil {
			logger.Error("Failed to remove temporary directory", "path", tmpDir, "error", err)
		}
	}()

	// Download from S3
	archivePath := filepath.Join(tmpDir, filepath.Base(key))
	logger.Info("Downloading from S3", "key", key, "target", archivePath)

	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	file, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, result.Body); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}
	file.Close()

	// Extract tar.gz
	logger.Info("Extracting archive", "archive", archivePath, "target", targetDir)
	if err := b.extractTarGz(archivePath, targetDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	logger.Info("Successfully restored directory", "directory", dirName)
	return nil
}

// matchesFilter checks if an S3 key matches the date filter
func (b *s3Backup) matchesFilter(key string, filter RestoreFilter) bool {
	// Parse year and month from key (format: "YYYY MM Month DD ...")
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return false
	}

	year := 0
	month := 0
	fmt.Sscanf(parts[0], "%d", &year)
	fmt.Sscanf(parts[1], "%d", &month)

	if year == 0 || month == 0 {
		return false
	}

	// Check lower bound
	if filter.FromYear > 0 {
		fromMonth := filter.FromMonth
		if fromMonth == 0 {
			fromMonth = 1 // Default to January
		}
		if year < filter.FromYear || (year == filter.FromYear && month < fromMonth) {
			return false
		}
	}

	// Check upper bound
	if filter.ToYear > 0 {
		toMonth := filter.ToMonth
		if toMonth == 0 {
			toMonth = 12 // Default to December
		}
		if year > filter.ToYear || (year == filter.ToYear && month > toMonth) {
			return false
		}
	}

	return true
}

// extractDirNameFromKey extracts directory name from S3 key
func (b *s3Backup) extractDirNameFromKey(key string) string {
	// Remove ".tar.gz" extension
	name := strings.TrimSuffix(key, ".tar.gz")
	// Remove " (X images, Y videos)" suffix
	if idx := strings.Index(name, " ("); idx != -1 {
		name = name[:idx]
	}
	return name
}

// extractTarGz extracts a tar.gz archive to a target directory
func (b *s3Backup) extractTarGz(archivePath, targetDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			// Restore file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}
