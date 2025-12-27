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

const (
	tempDirPrefix        = "/tmp/pics_tmp_%d"
	tempRestoreDirPrefix = "/tmp/pics_restore_%d"
)

// RestoreFilter defines the date range filter for restoring backups
type RestoreFilter struct {
	FromYear  int // 0 means no lower bound
	FromMonth int // 0 means January if FromYear is set
	ToYear    int // 0 means no upper bound
	ToMonth   int // 0 means December if ToYear is set
}

// S3ClientInterface defines the S3 operations we use
// The real *s3.Client naturally satisfies this interface (duck typing)
type S3ClientInterface interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// Backup defines the interface for backing up and restoring directories
type Backup interface {
	// BackupDirectories backs up all subdirectories in the source directory
	BackupDirectories(sourceDir, bucket string, maxConcurrent int) error
	// RestoreDirectories restores directories to target directory
	RestoreDirectories(bucket, targetDir string, filter RestoreFilter, maxConcurrent int) error
}

// s3Backup implements the Backup interface for AWS S3
type s3Backup struct {
	client     S3ClientInterface
	extensions Extensions
}

// NewS3Backup creates a new S3 Backup instance
func NewS3Backup(ctx context.Context) (Backup, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &s3Backup{
		client:     s3.NewFromConfig(cfg),
		extensions: NewExtensions(),
	}, nil
}

// Helper functions

// createTempDir creates a temporary directory with cleanup
func createTempDir(prefix string) (string, func(), error) {
	tmpDir := fmt.Sprintf(prefix, rand.Int())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup := func() {
		logger.Debug("Cleaning up temporary directory", "path", tmpDir)
		if err := os.RemoveAll(tmpDir); err != nil {
			logger.Error("Failed to remove temporary directory", "path", tmpDir, "error", err)
		}
	}

	return tmpDir, cleanup, nil
}

// runWorkerPool runs a worker pool and collects results
func runWorkerPool[T any](jobs []T, maxConcurrent int, workerFunc func(T) error) error {
	if len(jobs) == 0 {
		return nil
	}

	jobsChan := make(chan T, len(jobs))
	results := make(chan error, len(jobs))
	var wg sync.WaitGroup

	// Start workers
	for i := range maxConcurrent {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobsChan {
				results <- workerFunc(job)
			}
		}(i)
	}

	// Send jobs
	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	// Wait for completion
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
		return fmt.Errorf("completed with %d successes and %d failures", successCount, len(errors))
	}

	return nil
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

	// Run worker pool
	err = runWorkerPool(directories, maxConcurrent, func(dirName string) error {
		logger.Debug("Processing directory", "directory", dirName)
		if err := b.backupDirectory(sourceDir, dirName, bucket); err != nil {
			logger.Error("Failed to backup directory", "directory", dirName, "error", err)
			return fmt.Errorf("directory %s: %w", dirName, err)
		}
		return nil
	})

	if err != nil {
		logger.Error("Backup completed with errors", "error", err)
		return err
	}

	logger.Info("Backup completed successfully", "directories_backed_up", len(directories))
	return nil
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
	ctx := context.Background()
	dirPath := filepath.Join(sourceDir, dirName)

	// Count media files
	imageCount, videoCount, err := b.countMediaFiles(dirPath)
	if err != nil {
		return fmt.Errorf("failed to count media files: %w", err)
	}

	// Build S3 key with counts
	s3Key := fmt.Sprintf("%s (%d images, %d videos).tar.gz", dirName, imageCount, videoCount)

	// Create temporary directory
	tmpDir, cleanup, err := createTempDir(tempDirPrefix)
	if err != nil {
		return err
	}
	defer cleanup()

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
	headOutput, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	})

	if err == nil {
		// Object exists, check if hash matches
		remoteETag := b.extractETag(headOutput.ETag)
		if remoteETag == "" {
			return fmt.Errorf("S3 object exists but ETag is missing")
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

// extractETag safely extracts ETag value, removing quotes
func (b *s3Backup) extractETag(etag *string) string {
	if etag == nil || *etag == "" {
		return ""
	}
	etagValue := *etag
	// Remove quotes if present
	if len(etagValue) >= 2 && etagValue[0] == '"' && etagValue[len(etagValue)-1] == '"' {
		return etagValue[1 : len(etagValue)-1]
	}
	return etagValue
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

		// Copy file content and close immediately (not defer in loop)
		_, copyErr := io.Copy(tarWriter, f)
		f.Close()

		if copyErr != nil {
			return copyErr
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

	// Run worker pool
	err := runWorkerPool(objectsToRestore, maxConcurrent, func(obj types.Object) error {
		logger.Debug("Processing object", "key", *obj.Key)
		if err := b.restoreObject(bucket, targetDir, *obj.Key); err != nil {
			logger.Error("Failed to restore object", "key", *obj.Key, "error", err)
			return fmt.Errorf("object %s: %w", *obj.Key, err)
		}
		return nil
	})

	if err != nil {
		logger.Error("Restore completed with errors", "error", err)
		return err
	}

	logger.Info("Restore completed successfully", "directories_restored", len(objectsToRestore))
	return nil
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
	tmpDir, cleanup, err := createTempDir(tempRestoreDirPrefix)
	if err != nil {
		return err
	}
	defer cleanup()

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
