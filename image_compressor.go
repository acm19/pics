package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ImageCompressor defines the interface for compressing images
type ImageCompressor interface {
	// CompressDirectory compresses all JPEG images in a directory
	CompressDirectory(dir string, quality int) error
	// CompressFile compresses a single JPEG file
	CompressFile(path string, quality int) error
}

// jpegCompressor implements the ImageCompressor interface
type jpegCompressor struct{}

// NewImageCompressor creates a new ImageCompressor instance
func NewImageCompressor() ImageCompressor {
	return &jpegCompressor{}
}

// CompressDirectory compresses all JPEG files in a directory
func (c *jpegCompressor) CompressDirectory(dir string, quality int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	jpegFiles := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToUpper(filepath.Ext(entry.Name()))
		if ext == ".JPG" || ext == ".JPEG" {
			jpegFiles = append(jpegFiles, entry.Name())
		}
	}

	logger.Debug("Found JPEG files", "count", len(jpegFiles))
	for _, fileName := range jpegFiles {
		filePath := filepath.Join(dir, fileName)
		logger.Debug("Compressing file", "path", filePath)
		if err := c.CompressFile(filePath, quality); err != nil {
			return fmt.Errorf("failed to compress %s: %w", filePath, err)
		}
	}
	return nil
}

// CompressFile compresses a single JPEG file using jpegoptim (preserves EXIF)
func (c *jpegCompressor) CompressFile(path string, quality int) error {
	// jpegoptim preserves EXIF data and file modification time by default with -p flag
	cmd := exec.Command("jpegoptim", fmt.Sprintf("-m%d", quality), "-p", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jpegoptim failed for %s: %w, output: %s", path, err, output)
	}
	return nil
}
