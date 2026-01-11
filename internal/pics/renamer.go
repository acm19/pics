package pics

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/acm19/pics/internal/logger"
	"github.com/barasher/go-exiftool"
)

// fileFilter is a function that determines if a file should be renamed
type fileFilter func(filePath string) bool

// FileRenamer defines the interface for renaming files with patterns
type FileRenamer interface {
	// RenameFilesWithPattern renames files in a directory based on a filter and naming pattern.
	//
	// Files are renamed in place with sequential numbering: {baseName}_00001.ext, {baseName}_00002.ext, etc.
	// Only files matching the filter are renamed. Files are sorted by date (EXIF or modification time) first,
	// then by filename if dates are equal, to ensure consistent chronological ordering. File extensions are
	// normalised to lowercase.
	//
	// Parameters:
	//   - dir: The directory containing files to rename
	//   - baseName: The base name to use for renamed files (e.g., "vacation" produces "vacation_00001.jpg")
	//   - filter: A function that determines which files should be renamed
	//   - progressChan: Optional channel for progress events
	//
	// Returns:
	//   - int: The number of files that were renamed
	//   - error: An error if the directory cannot be read or files cannot be renamed
	RenameFilesWithPattern(dir, baseName string, filter fileFilter, progressChan chan<- ProgressEvent) (int, error)

	// MoveAndRenameFilesWithPattern moves files to a target directory and renames them with sequential numbering.
	//
	// Files matching the filter are moved from sourceDir to targetDir and renamed with the pattern
	// {baseName}_00001.ext, {baseName}_00002.ext, etc. Files are sorted by date (EXIF or modification time)
	// first, then by filename if dates are equal, to ensure consistent chronological ordering. File extensions
	// are normalised to lowercase.
	//
	// The target directory is created only if there are files to move. If no files match the filter,
	// the target directory is not created and the method returns successfully.
	//
	// Parameters:
	//   - sourceDir: The directory containing files to move
	//   - targetDir: The directory where files will be moved (created if needed and files exist)
	//   - baseName: The base name to use for renamed files
	//   - filter: A function that determines which files should be moved and renamed
	//   - progressChan: Optional channel for progress events
	//
	// Returns:
	//   - int: The number of files that were moved and renamed
	//   - error: An error if directories cannot be accessed or files cannot be moved
	MoveAndRenameFilesWithPattern(sourceDir, targetDir, baseName string, filter fileFilter, progressChan chan<- ProgressEvent) (int, error)
}

// fileRenamer implements the FileRenamer interface
type fileRenamer struct {
	dateExtractor *AggregatedFileDateExtractor
}

// NewFileRenamer creates a new FileRenamer instance
func NewFileRenamer(et *exiftool.Exiftool) FileRenamer {
	return &fileRenamer{
		dateExtractor: NewFileDateExtractor(et),
	}
}

// RenameFilesWithPattern renames files in a directory based on a filter and naming pattern
func (r *fileRenamer) RenameFilesWithPattern(dir, baseName string, filter fileFilter, progressChan chan<- ProgressEvent) (int, error) {
	return r.renameFilesWithPatternInDir(dir, dir, baseName, filter, progressChan)
}

// MoveAndRenameFilesWithPattern moves files to a target directory and renames them
func (r *fileRenamer) MoveAndRenameFilesWithPattern(sourceDir, targetDir, baseName string, filter fileFilter, progressChan chan<- ProgressEvent) (int, error) {
	return r.renameFilesWithPatternInDir(sourceDir, targetDir, baseName, filter, progressChan)
}

// fileWithDate holds a file path and its extracted date
type fileWithDate struct {
	path string
	date time.Time
}

// renameFilesWithPatternInDir is the internal implementation
func (r *fileRenamer) renameFilesWithPatternInDir(sourceDir, targetDir, baseName string, filter fileFilter, progressChan chan<- ProgressEvent) (int, error) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	// Collect files matching the filter with their dates
	var filesWithDates []fileWithDate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(sourceDir, entry.Name())
		if filter(filePath) {
			// Extract date for this file
			date, err := r.dateExtractor.GetFileDate(filePath)
			if err != nil {
				logger.Warn("Failed to extract date, using zero time", "file", filePath, "error", err)
				date = time.Time{} // Use zero time as fallback
			}
			filesWithDates = append(filesWithDates, fileWithDate{
				path: filePath,
				date: date,
			})
		}
	}

	// Nothing to rename
	if len(filesWithDates) == 0 {
		return 0, nil
	}

	// Create target directory only if there are files to move
	if sourceDir != targetDir {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return 0, fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	// Sort files by date (oldest first), then by filename if dates are equal
	sort.Slice(filesWithDates, func(i, j int) bool {
		if filesWithDates[i].date.Equal(filesWithDates[j].date) {
			// If dates are equal, sort by filename
			return filesWithDates[i].path < filesWithDates[j].path
		}
		return filesWithDates[i].date.Before(filesWithDates[j].date)
	})

	// Rename each file with sequential numbering
	totalFiles := len(filesWithDates)
	for i, fileData := range filesWithDates {
		// Emit progress event
		if progressChan != nil {
			select {
			case progressChan <- ProgressEvent{
				Stage:   "renaming",
				Current: i + 1,
				Total:   totalFiles,
				Message: fmt.Sprintf("Renaming file %d of %d", i+1, totalFiles),
				File:    fileData.path,
			}:
			default:
				logger.Debug("Progress event dropped (channel full)", "stage", "renaming")
			}
		}

		ext := strings.ToLower(filepath.Ext(fileData.path))
		newFileName := fmt.Sprintf("%s_%05d%s", baseName, i+1, ext)
		newFilePath := filepath.Join(targetDir, newFileName)

		if err := os.Rename(fileData.path, newFilePath); err != nil {
			return 0, fmt.Errorf("failed to rename %s to %s: %w", fileData.path, newFilePath, err)
		}
	}

	return len(filesWithDates), nil
}
