package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MediaParser defines the interface for parsing and organising media files
type MediaParser interface {
	// Parse processes media files from source to target directory
	Parse(sourceDir, targetDir string, opts ParseOptions) error
}

// ParseOptions holds configuration options for parsing
type ParseOptions struct {
	// CompressJPEGs enables JPEG compression
	CompressJPEGs bool
	// JPEGQuality is the quality level for JPEG compression (0-100)
	JPEGQuality int
	// TempDirName is the name of the temporary directory to use
	TempDirName string
	// MaxConcurrency is the maximum number of files to process concurrently (0 = unlimited)
	MaxConcurrency int
}

// DefaultParseOptions returns the default parsing options
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		CompressJPEGs:  true,
		JPEGQuality:    50,
		TempDirName:    "tmp_image",
		MaxConcurrency: 100,
	}
}

// mediaParser implements the MediaParser interface
type mediaParser struct {
	compressor ImageCompressor
	organiser  FileOrganiser
	extensions Extensions
}

// NewMediaParser creates a new MediaParser instance
func NewMediaParser() MediaParser {
	return &mediaParser{
		compressor: NewImageCompressor(),
		organiser:  NewFileOrganiser(),
		extensions: NewExtensions(),
	}
}

// Parse processes media files from source to target directory
func (p *mediaParser) Parse(sourceDir, targetDir string, opts ParseOptions) error {
	sourceDir = strings.TrimSuffix(sourceDir, "/")
	targetDir = strings.TrimSuffix(targetDir, "/")
	tmpTarget := filepath.Join(targetDir, opts.TempDirName)

	logger.Info("Creating temporary directory", "path", tmpTarget)
	if err := os.MkdirAll(tmpTarget, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpTarget)

	logger.Info("Processing media files (copy and compress)", "source", sourceDir, "target", tmpTarget)
	processStart := time.Now()
	if err := p.copyAndCompressFiles(sourceDir, tmpTarget, opts); err != nil {
		return fmt.Errorf("failed to process media files: %w", err)
	}
	processDuration := time.Since(processStart)
	logger.Info("Processing completed", "duration_seconds", processDuration.Seconds())

	logger.Info("Organising files by date")
	if err := p.organiser.OrganiseByDate(tmpTarget, targetDir); err != nil {
		return fmt.Errorf("failed to organise by date: %w", err)
	}

	// Remove temporary directory before organising (all files have been moved to date-based directories)
	logger.Debug("Removing temporary directory", "path", tmpTarget)
	if err := os.RemoveAll(tmpTarget); err != nil {
		return fmt.Errorf("failed to remove temp directory: %w", err)
	}

	logger.Info("Organising videos and renaming images")
	if err := p.organiser.OrganiseVideosAndRenameImages(targetDir); err != nil {
		return fmt.Errorf("failed to organise videos and rename images: %w", err)
	}

	logger.Info("Processing complete")
	return nil
}

type fileToProcess struct {
	srcPath  string
	destPath string
	isJPEG   bool
}

// copyAndCompressFiles copies and optionally compresses files in parallel using a worker pool
func (p *mediaParser) copyAndCompressFiles(sourceDir, tmpTarget string, opts ParseOptions) error {
	// Determine number of workers
	numWorkers := opts.MaxConcurrency
	if numWorkers <= 0 {
		numWorkers = 100 // Default if unlimited
	}

	jobs := make(chan fileToProcess, numWorkers)
	var wg sync.WaitGroup
	errChan := make(chan error, numWorkers)

	// Start worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go p.processFileWorker(jobs, errChan, opts, &wg)
	}

	// Discover and send files to workers
	go p.discoverFiles(sourceDir, tmpTarget, jobs)

	wg.Wait()
	close(errChan)

	// Collect all errors from workers
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	// Return first error if any occurred
	if len(errors) > 0 {
		if len(errors) > 1 {
			logger.Error("Multiple errors occurred during processing", "error_count", len(errors))
			for i, err := range errors {
				logger.Error("Processing error", "index", i+1, "error", err)
			}
		}
		return errors[0]
	}

	return nil
}

// processFileWorker processes files from the jobs channel
func (p *mediaParser) processFileWorker(jobs <-chan fileToProcess, errChan chan<- error, opts ParseOptions, wg *sync.WaitGroup) {
	defer wg.Done()
	for file := range jobs {
		logger.Debug("Copying file", "from", file.srcPath, "to", file.destPath)
		if err := copyFilePreserveTime(file.srcPath, file.destPath); err != nil {
			errChan <- fmt.Errorf("failed to copy %s: %w", file.srcPath, err)
			continue
		}

		if file.isJPEG && opts.CompressJPEGs {
			logger.Debug("Compressing file", "path", file.destPath)
			if err := p.compressor.CompressFile(file.destPath, opts.JPEGQuality); err != nil {
				errChan <- fmt.Errorf("failed to compress %s: %w", file.destPath, err)
				continue
			}
		}

		logger.Debug("Finished processing file", "path", file.destPath)
	}
}

// discoverFiles walks directories recursively and sends files to the jobs channel
func (p *mediaParser) discoverFiles(sourceDir, tmpTarget string, jobs chan<- fileToProcess) {
	defer close(jobs)
	logger.Info("Discovering files to process", "source", sourceDir)

	filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Debug("Error accessing path", "path", path, "error", err)
			return err
		}

		// Skip dot files and dot directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if p.extensions.IsSupported(path) {
			// Calculate relative path from source directory for prefixing
			relPath, err := filepath.Rel(sourceDir, path)
			if err != nil {
				logger.Debug("Failed to calculate relative path", "path", path, "error", err)
				return err
			}

			// Use directory structure as prefix, replacing path separators with dashes
			prefix := strings.ReplaceAll(filepath.Dir(relPath), string(filepath.Separator), "-")
			if prefix == "." {
				prefix = "root"
			}

			destPath := filepath.Join(tmpTarget, fmt.Sprintf("%s-%s", prefix, filepath.Base(path)))
			logger.Debug("Discovered file", "path", path, "dest", destPath)

			jobs <- fileToProcess{
				srcPath:  path,
				destPath: destPath,
				isJPEG:   p.extensions.IsJPEG(path),
			}
		}
		return nil
	})
}

// copyFilePreserveTime copies a file and preserves its modification time
func copyFilePreserveTime(src, dst string) error {
	logger.Debug("Starting file copy", "from", src, "to", dst)

	srcInfo, err := os.Stat(src)
	if err != nil {
		logger.Debug("Failed to stat source file", "file", src, "error", err)
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		logger.Debug("Failed to open source file", "file", src, "error", err)
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		logger.Debug("Failed to create destination file", "file", dst, "error", err)
		return err
	}
	defer dstFile.Close()

	bytesWritten, err := io.Copy(dstFile, srcFile)
	if err != nil {
		logger.Debug("Failed to copy file contents", "from", src, "to", dst, "error", err)
		return err
	}

	logger.Debug("File copied successfully", "from", src, "to", dst, "bytes", bytesWritten)

	if err := os.Chtimes(dst, time.Now(), srcInfo.ModTime()); err != nil {
		logger.Debug("Failed to preserve modification time", "file", dst, "error", err)
		return err
	}

	logger.Debug("Modification time preserved", "file", dst, "modTime", srcInfo.ModTime())
	return nil
}
