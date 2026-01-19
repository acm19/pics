package pics

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/acm19/pics/internal/logger"
	"github.com/barasher/go-exiftool"
)

const (
	// ExifOriginalFileName is the EXIF field name for storing the original filename
	ExifOriginalFileName = "OriginalFileName"
)

// ExifWriter defines the interface for writing EXIF metadata
type ExifWriter interface {
	// WriteOriginalFileNameIfMissing writes the original filename to EXIF metadata
	// if it doesn't already exist. Only processes image files (JPG, JPEG, HEIC, PNG).
	// Returns true if the field was written, false if it already exists or file is not an image.
	WriteOriginalFileNameIfMissing(filePath string, originalFileName string) (bool, error)
}

// exifWriter implements the ExifWriter interface
type exifWriter struct {
	et         *exiftool.Exiftool
	extensions Extensions
}

// NewExifWriter creates a new ExifWriter instance
func NewExifWriter(et *exiftool.Exiftool) ExifWriter {
	return &exifWriter{
		et:         et,
		extensions: NewExtensions(),
	}
}

// WriteOriginalFileNameIfMissing writes the original filename to EXIF metadata if it doesn't already exist
func (w *exifWriter) WriteOriginalFileNameIfMissing(filePath string, originalFileName string) (bool, error) {
	if w.et == nil {
		return false, fmt.Errorf("exiftool not initialised")
	}

	// Only process image files - skip videos as they don't support this field well
	if !w.extensions.IsImage(filePath) {
		logger.Debug("Skipping EXIF write for non-image file", "file", filepath.Base(filePath))
		return false, nil
	}

	// Check if the field already exists
	fileInfos := w.et.ExtractMetadata(filePath)
	if len(fileInfos) > 0 && fileInfos[0].Err == nil {
		if _, err := fileInfos[0].GetString(ExifOriginalFileName); err == nil {
			logger.Debug("OriginalFileName already exists, skipping", "file", filepath.Base(filePath))
			return false, nil
		}
	}

	// Use exiftool command-line to write the OriginalFileName tag
	// -overwrite_original prevents creating backup files
	// -P preserves the file modification date/time
	cmd := exec.Command("exiftool",
		"-"+ExifOriginalFileName+"="+originalFileName,
		"-overwrite_original",
		"-P",
		filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to write %s: %w (output: %s)", ExifOriginalFileName, err, string(output))
	}

	logger.Debug("Wrote OriginalFileName to EXIF", "file", originalFileName)
	return true, nil
}
