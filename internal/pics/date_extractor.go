package pics

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/acm19/pics/internal/logger"
	"github.com/barasher/go-exiftool"
)

// fileDateExtractor defines the interface for extracting file dates
type fileDateExtractor interface {
	getFileDate(filePath string) (time.Time, error)
	name() string
}

// modTimeExtractor extracts date from file modification time
type modTimeExtractor struct{}

func newModTimeExtractor() *modTimeExtractor {
	return &modTimeExtractor{}
}

func (e *modTimeExtractor) name() string {
	return "ModTime"
}

func (e *modTimeExtractor) getFileDate(filePath string) (time.Time, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}
	logger.Debug("Using file modification time", "file", filepath.Base(filePath), "modTime", info.ModTime())
	return info.ModTime(), nil
}

// exifDateExtractor extracts date from EXIF metadata
type exifDateExtractor struct{}

func newExifDateExtractor() *exifDateExtractor {
	return &exifDateExtractor{}
}

func (e *exifDateExtractor) name() string {
	return "EXIF"
}

func (e *exifDateExtractor) getFileDate(filePath string) (time.Time, error) {
	et, err := exiftool.NewExiftool()
	if err != nil {
		return time.Time{}, err
	}
	defer et.Close()

	fileInfos := et.ExtractMetadata(filePath)
	if len(fileInfos) == 0 {
		return time.Time{}, fmt.Errorf("no metadata found")
	}

	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
		return time.Time{}, fileInfo.Err
	}

	// Try date fields in order of preference: CreationDate first, then CreateDate
	dateFields := []string{"CreationDate", "CreateDate"}
	for _, field := range dateFields {
		if val, err := fileInfo.GetString(field); err == nil {
			logger.Debug("Using EXIF date field", "file", filepath.Base(filePath), "field", field, "date", val)

			// Parse the EXIF date string (format: "2006:01:02 15:04:05")
			parsedTime, err := time.Parse("2006:01:02 15:04:05", val)
			if err != nil {
				logger.Debug("Failed to parse EXIF date", "file", filePath, "date", val, "error", err)
				return time.Time{}, err
			}
			return parsedTime, nil
		}
	}

	// No valid EXIF date found
	return time.Time{}, fmt.Errorf("no EXIF date field found")
}

// AggregatedFileDateExtractor iterates through multiple extractors until one succeeds
type AggregatedFileDateExtractor struct {
	extractors []fileDateExtractor
}

// NewFileDateExtractor creates a new AggregatedFileDateExtractor with EXIF and ModTime extractors
//
// Prioritises extracting the dates from the EXIF metadata in the following
// order:
//
//   - CreationDate: because modified iPhone videos keep the original date in
//     this field.
//   - CreateDate: holds the date when the image/video was created.
//   - ModTime: if nothing else works falls back to modification time.
func NewFileDateExtractor() *AggregatedFileDateExtractor {
	return &AggregatedFileDateExtractor{
		extractors: []fileDateExtractor{
			newExifDateExtractor(),
			newModTimeExtractor(),
		},
	}
}

// GetFileDate extracts the creation date by trying each extractor in order
// Works for both images (JPG, HEIC) and videos (MOV)
func (e *AggregatedFileDateExtractor) GetFileDate(filePath string) (time.Time, error) {
	for _, extractor := range e.extractors {
		date, err := extractor.getFileDate(filePath)
		if err == nil && !date.IsZero() {
			return date, nil
		}
		if err != nil {
			logger.Debug("Extractor failed, trying next", "extractor", extractor.name(), "file", filepath.Base(filePath), "error", err)
		}
	}

	return time.Time{}, fmt.Errorf("all extractors failed for file: %s", filePath)
}
