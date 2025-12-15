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
	// ValidateDirectories checks if source and target directories exist
	ValidateDirectories(sourceDir, targetDir string) error
	// GetFileCount returns the number of files in a directory recursively
	GetFileCount(dir string) (int, error)
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
}

// NewMediaParser creates a new MediaParser instance
func NewMediaParser() MediaParser {
	return &mediaParser{
		compressor: NewImageCompressor(),
		organiser:  NewFileOrganiser(),
	}
}

// ValidateDirectories checks if source and target directories exist
func (p *mediaParser) ValidateDirectories(sourceDir, targetDir string) error {
	if info, err := os.Stat(sourceDir); err != nil || !info.IsDir() {
		return fmt.Errorf("SOURCE_DIR is not a valid directory: %s", sourceDir)
	}
	if info, err := os.Stat(targetDir); err != nil || !info.IsDir() {
		return fmt.Errorf("TARGET_DIR is not a valid directory: %s", targetDir)
	}
	return nil
}

// Parse processes media files from source to target directory
func (p *mediaParser) Parse(sourceDir, targetDir string, opts ParseOptions) error {
	if err := p.ValidateDirectories(sourceDir, targetDir); err != nil {
		return err
	}
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

	logger.Info("Organising videos and renaming JPGs")
	if err := p.organiser.OrganiseVideosAndRenameJPGs(targetDir); err != nil {
		return fmt.Errorf("failed to organise videos and rename JPGs: %w", err)
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

	if err := <-errChan; err != nil {
		return err
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

// discoverFiles walks directories and sends files to the jobs channel
func (p *mediaParser) discoverFiles(sourceDir, tmpTarget string, jobs chan<- fileToProcess) {
	defer close(jobs)

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		logger.Error("Failed to read source directory", "error", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		prefix := entry.Name()
		subDir := filepath.Join(sourceDir, entry.Name())
		logger.Debug("Processing subdirectory", "dir", prefix)

		filepath.Walk(subDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			ext := strings.ToUpper(filepath.Ext(path))
			if ext == ".MOV" || ext == ".JPG" {
				destPath := filepath.Join(tmpTarget, fmt.Sprintf("%s-%s", prefix, filepath.Base(path)))
				jobs <- fileToProcess{
					srcPath:  path,
					destPath: destPath,
					isJPEG:   ext == ".JPG",
				}
			}
			return nil
		})
	}
}

// copyFilePreserveTime copies a file and preserves its modification time
func copyFilePreserveTime(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return os.Chtimes(dst, time.Now(), srcInfo.ModTime())
}

// GetFileCount returns the number of files in a directory recursively
func (p *mediaParser) GetFileCount(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}
