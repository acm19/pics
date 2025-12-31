package pics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/acm19/pics/internal/logger"
)

// FileOrganiser defines the interface for organising files
type FileOrganiser interface {
	// OrganiseByDate moves files to date-based directories
	OrganiseByDate(sourceDir, targetDir string, progressChan chan<- ProgressEvent) error
	// OrganiseVideosAndRenameImages organises videos into subdirectories and renames images sequentially
	OrganiseVideosAndRenameImages(targetDir string, progressChan chan<- ProgressEvent) error
}

// fileOrganiser implements the FileOrganiser interface
type fileOrganiser struct {
	dateExtractor *AggregatedFileDateExtractor
	extensions    Extensions
	fileRenamer   FileRenamer
}

// NewFileOrganiser creates a new FileOrganiser instance
func NewFileOrganiser() FileOrganiser {
	return &fileOrganiser{
		dateExtractor: NewFileDateExtractor(),
		extensions:    NewExtensions(),
		fileRenamer:   NewFileRenamer(),
	}
}

// NewFileOrganiserWithPaths creates a new FileOrganiser with custom binary paths
func NewFileOrganiserWithPaths(exiftoolPath string) FileOrganiser {
	return &fileOrganiser{
		dateExtractor: NewFileDateExtractorWithPath(exiftoolPath),
		extensions:    NewExtensions(),
		fileRenamer:   NewFileRenamer(),
	}
}

// OrganiseByDate moves files to date-based directories
func (o *fileOrganiser) OrganiseByDate(sourceDir, targetDir string, progressChan chan<- ProgressEvent) error {
	logger.Info("OrganiseByDate started", "sourceDir", sourceDir, "targetDir", targetDir)

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	logger.Info("Directory read complete", "entries", len(entries))

	// Count total files
	totalFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			totalFiles++
		}
	}
	logger.Debug("Counted files", "totalFiles", totalFiles)

	current := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(sourceDir, entry.Name())
		current++

		// Emit progress event
		if progressChan != nil {
			select {
			case progressChan <- ProgressEvent{
				Stage:   "organising",
				Current: current,
				Total:   totalFiles,
				Message: fmt.Sprintf("Organising file %d of %d", current, totalFiles),
				File:    filePath,
			}:
			default:
				logger.Debug("Progress event dropped (channel full)", "stage", "organising")
			}
		}

		// Get file date from EXIF if available, otherwise use ModTime
		logger.Debug("Extracting date", "file", entry.Name(), "current", current, "total", totalFiles)
		fileDate, err := o.dateExtractor.GetFileDate(filePath)
		if err != nil {
			logger.Error("Failed to get file date", "file", entry.Name(), "error", err)
			return err
		}
		logger.Debug("Date extracted", "file", entry.Name(), "date", fileDate)

		dirName := fileDate.Format("2006 01 January 02")
		destDir := filepath.Join(targetDir, dirName)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
		if err := os.Rename(filePath, filepath.Join(destDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// OrganiseVideosAndRenameImages organises videos into subdirectories and renames images sequentially
func (o *fileOrganiser) OrganiseVideosAndRenameImages(targetDir string, progressChan chan<- ProgressEvent) error {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}

	// Count total directories
	totalDirs := 0
	for _, entry := range entries {
		if entry.IsDir() {
			totalDirs++
		}
	}

	current := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(targetDir, entry.Name())
		current++

		// Emit progress event
		if progressChan != nil {
			select {
			case progressChan <- ProgressEvent{
				Stage:   "organising",
				Current: current,
				Total:   totalDirs,
				Message: fmt.Sprintf("Organising directory %d of %d", current, totalDirs),
				File:    dirPath,
			}:
			default:
				logger.Debug("Progress event dropped (channel full)", "stage", "organising")
			}
		}

		logger.Debug("Organising file %s/%s", dirPath, entry.Name())
		if err := o.organiseVideos(dirPath, entry.Name(), progressChan); err != nil {
			return err
		}
		if err := o.renameImages(dirPath, entry.Name(), progressChan); err != nil {
			return err
		}
	}
	return nil
}

// organiseVideos moves video files to a videos subdirectory and renames them sequentially
func (o *fileOrganiser) organiseVideos(dir string, dirName string, progressChan chan<- ProgressEvent) error {
	parts := strings.Fields(dirName)
	if len(parts) != 4 {
		return fmt.Errorf("unexpected directory name format: %s", dirName)
	}
	videosName := strings.Join(parts, "_")
	videosDir := filepath.Join(dir, "videos")
	_, err := o.fileRenamer.MoveAndRenameFilesWithPattern(dir, videosDir, videosName, o.extensions.IsVideo, progressChan)
	return err
}

// renameImages renames image files with a sequential pattern
func (o *fileOrganiser) renameImages(dir, dirName string, progressChan chan<- ProgressEvent) error {
	parts := strings.Fields(dirName)
	if len(parts) != 4 {
		return fmt.Errorf("unexpected directory name format: %s", dirName)
	}
	picsName := strings.Join(parts, "_")
	_, err := o.fileRenamer.RenameFilesWithPattern(dir, picsName, o.extensions.IsImage, progressChan)
	return err
}
