package pics

import (
	"fmt"
	"os"
	"os/exec"
)

// ImageCompressor defines the interface for compressing images
type ImageCompressor interface {
	// CompressFile compresses a single JPEG file
	CompressFile(path string, quality int) error
}

// jpegCompressor implements the ImageCompressor interface
type jpegCompressor struct{}

// NewImageCompressor creates a new ImageCompressor instance
func NewImageCompressor() ImageCompressor {
	return &jpegCompressor{}
}

// CompressFile compresses a single JPEG file using jpegoptim (preserves EXIF)
func (c *jpegCompressor) CompressFile(path string, quality int) error {
	// Check if file exists first
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}

	// jpegoptim preserves EXIF data and file modification time by default with -p flag
	cmd := exec.Command("jpegoptim", fmt.Sprintf("-m%d", quality), "-p", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jpegoptim failed for %s: %w, output: %s", path, err, output)
	}
	return nil
}
