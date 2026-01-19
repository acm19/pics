package pics

import (
	"path/filepath"
	"slices"
	"strings"
)

// Extensions defines the interface for file extension operations.
type Extensions interface {
	// IsImage returns true if the file extension is a supported image format.
	IsImage(filePath string) bool
	// IsVideo returns true if the file extension is a supported video format.
	IsVideo(filePath string) bool
	// IsSupported returns true if the file extension is any supported media format.
	IsSupported(filePath string) bool
	// IsJPEG returns true if the file extension is JPEG (jpg or jpeg).
	IsJPEG(filePath string) bool
}

// extensions implements the Extensions interface.
type extensions struct {
	imageExts []string
	videoExts []string
}

// NewExtensions creates a new Extensions instance.
func NewExtensions() Extensions {
	return &extensions{
		imageExts: []string{".jpg", ".jpeg", ".heic", ".png"},
		videoExts: []string{
			".mov",   // QuickTime
			".mp4",   // MPEG-4
			".avi",   // Audio Video Interleave
			".mkv",   // Matroska Video Container
			".webm",  // WebM Video
			".flv",   // Flash Video
			".wmv",   // Windows Media Video
			".m4v",   // MPEG-4 Video (Apple)
			".3gp",   // 3GPP (mobile)
			".m2ts",  // MPEG-2 Transport Stream
			".mts",   // MPEG-2 Transport Stream
			".ogv",   // Ogg Theora Video
			".ts",    // MPEG-2 Transport Stream
		},
	}
}

// IsImage returns true if the file extension is a supported image format.
func (e *extensions) IsImage(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return slices.Contains(e.imageExts, ext)
}

// IsVideo returns true if the file extension is a supported video format.
func (e *extensions) IsVideo(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return slices.Contains(e.videoExts, ext)
}

// IsSupported returns true if the file extension is any supported media format.
func (e *extensions) IsSupported(filePath string) bool {
	return e.IsImage(filePath) || e.IsVideo(filePath)
}

// IsJPEG returns true if the file extension is JPEG (jpg or jpeg).
func (e *extensions) IsJPEG(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".jpg" || ext == ".jpeg"
}
