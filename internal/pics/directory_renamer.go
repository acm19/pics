package pics

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/acm19/pics/internal/logger"
	"github.com/barasher/go-exiftool"
)

// DirectoryRenamer defines the interface for renaming date-based directories
type DirectoryRenamer interface {
	// RenameDirectory renames a date-based directory and all images inside it
	RenameDirectory(directory, newName string) error
}

// directoryRenamer implements the DirectoryRenamer interface
type directoryRenamer struct {
	extensions  Extensions
	fileRenamer FileRenamer
}

// NewDirectoryRenamer creates a new DirectoryRenamer instance
func NewDirectoryRenamer(et *exiftool.Exiftool) DirectoryRenamer {
	return &directoryRenamer{
		extensions:  NewExtensions(),
		fileRenamer: NewFileRenamer(et),
	}
}

// RenameDirectory renames a date-based directory and all images inside it
func (r *directoryRenamer) RenameDirectory(directory, newName string) error {
	// Clean the path to remove trailing slashes and normalize
	directory = filepath.Clean(directory)

	// Check if directory exists
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", directory)
	}

	// Convert to absolute path to ensure correct parent directory
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Extract base name and parse date
	baseName := filepath.Base(absDir)
	parts := strings.Fields(baseName)

	// Expect at least 4 parts: YYYY MM Month DD
	if len(parts) < 4 {
		return fmt.Errorf("directory name does not match expected format (YYYY MM Month DD [name]): %s", baseName)
	}

	// Validate year and month are numeric
	year, err := strconv.Atoi(parts[0])
	if err != nil || year < 1000 || year > 9999 {
		return fmt.Errorf("invalid year in directory name: %s", parts[0])
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return fmt.Errorf("invalid month in directory name: %s", parts[1])
	}

	// Build new directory name: date + new name
	dateParts := parts[:4]
	newDirName := strings.Join(dateParts, " ")
	if newName != "" {
		newDirName = newDirName + " " + newName
	}

	// Build full path for new directory
	parentDir := filepath.Dir(absDir)
	newDirPath := filepath.Join(parentDir, newDirName)

	logger.Debug("Rename paths", "original", directory, "absolute", absDir, "parent", parentDir, "new_name", newDirName, "new_path", newDirPath)

	// If the new path is the same as old, no directory rename needed
	if absDir == newDirPath {
		logger.Info("Directory name unchanged, updating images only")
	} else {
		// Check if target directory already exists
		if _, err := os.Stat(newDirPath); err == nil {
			return fmt.Errorf("target directory already exists: %s", newDirPath)
		}

		logger.Info("Renaming directory", "from", absDir, "to", newDirPath)
	}

	// Convert directory name to base name for file renaming
	newBaseName := strings.ReplaceAll(newDirName, " ", "_")

	// Rename image files first (before moving directory)
	if err := r.renameImages(absDir, newBaseName); err != nil {
		return err
	}

	// Rename videos in videos subdirectory if it exists
	if err := r.renameVideos(absDir, newBaseName); err != nil {
		return err
	}

	// Rename the directory if needed
	if err := r.renameDir(absDir, newDirPath); err != nil {
		return err
	}

	return nil
}

// renameImages renames all image files in the directory
func (r *directoryRenamer) renameImages(absDir, newBaseName string) error {
	imageCount, err := r.fileRenamer.RenameFilesWithPattern(absDir, newBaseName, r.extensions.IsImage, nil)
	if err != nil {
		return err
	}

	if imageCount > 0 {
		logger.Info("Renaming images", "count", imageCount, "pattern", newBaseName)
	}

	return nil
}

// renameVideos renames all video files in the videos subdirectory
func (r *directoryRenamer) renameVideos(absDir, newBaseName string) error {
	videosDir := filepath.Join(absDir, "videos")
	info, err := os.Stat(videosDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	videoCount, err := r.fileRenamer.MoveAndRenameFilesWithPattern(videosDir, videosDir, newBaseName, r.extensions.IsVideo, nil)
	if err != nil {
		return err
	}

	if videoCount > 0 {
		logger.Info("Renaming videos", "count", videoCount, "pattern", newBaseName)
	}

	return nil
}

// renameDir renames the directory itself
func (r *directoryRenamer) renameDir(absDir, newDirPath string) error {
	if absDir == newDirPath {
		return nil
	}

	if err := os.Rename(absDir, newDirPath); err != nil {
		return fmt.Errorf("failed to rename directory: %w", err)
	}
	logger.Info("Directory renamed successfully", "new_path", newDirPath)

	return nil
}
