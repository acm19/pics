package main

import (
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// CompressFile re-encodes a single JPEG file at the specified quality
func (c *jpegCompressor) CompressFile(path string, quality int) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	modTime := info.ModTime()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		return err
	}
	file.Close()

	tmpPath := path + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(outFile, img, opts); err != nil {
		outFile.Close()
		return err
	}
	outFile.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	return os.Chtimes(path, time.Now(), modTime)
}
