package pics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileStats defines the interface for file and directory statistics
type FileStats interface {
	// ValidateDirectories checks if source and target directories exist
	ValidateDirectories(sourceDir, targetDir string) error
	// GetFileCount returns the number of supported media files in a directory recursively
	GetFileCount(dir string) (int, error)
	// GetUnsupportedFiles returns a list of unsupported files in a directory recursively
	GetUnsupportedFiles(dir string) ([]string, error)
}

// fileStats implements the FileStats interface
type fileStats struct {
	extensions Extensions
}

// NewFileStats creates a new FileStats instance
func NewFileStats() FileStats {
	return &fileStats{
		extensions: NewExtensions(),
	}
}

// ValidateDirectories checks if source and target directories exist
func (f *fileStats) ValidateDirectories(sourceDir, targetDir string) error {
	if info, err := os.Stat(sourceDir); err != nil || !info.IsDir() {
		return fmt.Errorf("SOURCE_DIR is not a valid directory: %s", sourceDir)
	}
	if info, err := os.Stat(targetDir); err != nil || !info.IsDir() {
		return fmt.Errorf("TARGET_DIR is not a valid directory: %s", targetDir)
	}
	return nil
}

// GetFileCount counts all supported media files in a directory tree, excluding dot files
func (f *fileStats) GetFileCount(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip dot files and dot directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && f.extensions.IsSupported(path) {
			count++
		}
		return nil
	})
	return count, err
}

// GetUnsupportedFiles returns a list of unsupported files in a directory tree, excluding dot files
func (f *fileStats) GetUnsupportedFiles(dir string) ([]string, error) {
	var unsupported []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip dot files and dot directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && !f.extensions.IsSupported(path) {
			unsupported = append(unsupported, path)
		}
		return nil
	})
	return unsupported, err
}
