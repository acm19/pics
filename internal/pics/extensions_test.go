package pics

import (
	"testing"
)

func TestExtensions_IsImage(t *testing.T) {
	ext := NewExtensions()

	tests := []struct {
		filePath string
		expected bool
	}{
		{"photo.jpg", true},
		{"photo.JPG", true},
		{"photo.jpeg", true},
		{"photo.JPEG", true},
		{"photo.heic", true},
		{"photo.HEIC", true},
		{"photo.png", true},
		{"photo.PNG", true},
		{"video.mov", false},
		{"video.mp4", false},
		{"document.txt", false},
		{"/path/to/image.jpg", true},
		{"/path/to/image.HEIC", true},
		{"/path/to/image.png", true},
	}

	for _, tt := range tests {
		result := ext.IsImage(tt.filePath)
		if result != tt.expected {
			t.Errorf("IsImage(%s) = %v, expected %v", tt.filePath, result, tt.expected)
		}
	}
}

func TestExtensions_IsVideo(t *testing.T) {
	ext := NewExtensions()

	tests := []struct {
		filePath string
		expected bool
	}{
		{"video.mov", true},
		{"video.MOV", true},
		{"video.mp4", true},
		{"video.MP4", true},
		{"photo.jpg", false},
		{"photo.heic", false},
		{"document.txt", false},
		{"file.avi", true},
		{"/path/to/video.mov", true},
		{"/path/to/video.MP4", true},
	}

	for _, tt := range tests {
		result := ext.IsVideo(tt.filePath)
		if result != tt.expected {
			t.Errorf("IsVideo(%s) = %v, expected %v", tt.filePath, result, tt.expected)
		}
	}
}

func TestExtensions_IsSupported(t *testing.T) {
	ext := NewExtensions()

	tests := []struct {
		filePath string
		expected bool
	}{
		// Images
		{"photo.jpg", true},
		{"photo.JPEG", true},
		{"photo.heic", true},
		{"photo.png", true},
		{"photo.PNG", true},
		// Videos
		{"video.mov", true},
		{"video.mp4", true},
		{"video.MP4", true},
		// Unsupported
		{"document.txt", false},
		{"file.avi", true},
		{"file.gif", false},
		{"/path/to/supported.jpg", true},
		{"/path/to/supported.png", true},
		{"/path/to/unsupported.pdf", false},
	}

	for _, tt := range tests {
		result := ext.IsSupported(tt.filePath)
		if result != tt.expected {
			t.Errorf("IsSupported(%s) = %v, expected %v", tt.filePath, result, tt.expected)
		}
	}
}

func TestExtensions_IsJPEG(t *testing.T) {
	ext := NewExtensions()

	tests := []struct {
		filePath string
		expected bool
	}{
		{"photo.jpg", true},
		{"photo.JPG", true},
		{"photo.jpeg", true},
		{"photo.JPEG", true},
		{"photo.heic", false},
		{"video.mov", false},
		{"document.txt", false},
		{"/path/to/image.jpg", true},
		{"/path/to/image.JPEG", true},
		{"/path/to/image.heic", false},
	}

	for _, tt := range tests {
		result := ext.IsJPEG(tt.filePath)
		if result != tt.expected {
			t.Errorf("IsJPEG(%s) = %v, expected %v", tt.filePath, result, tt.expected)
		}
	}
}

func TestExtensions_CaseInsensitive(t *testing.T) {
	ext := NewExtensions()

	// Test that extension checks are case-insensitive
	testCases := []string{
		"file.jpg", "file.JPG", "file.JpG", "file.jpG",
		"file.mov", "file.MOV", "file.MoV", "file.moV",
	}

	for _, filePath := range testCases {
		if !ext.IsSupported(filePath) {
			t.Errorf("IsSupported(%s) should be true (case-insensitive)", filePath)
		}
	}
}

func TestExtensions_NoExtension(t *testing.T) {
	ext := NewExtensions()

	tests := []string{
		"filename",
		"file_without_extension",
		"/path/to/file",
	}

	for _, filePath := range tests {
		if ext.IsImage(filePath) {
			t.Errorf("IsImage(%s) should be false for file without extension", filePath)
		}
		if ext.IsVideo(filePath) {
			t.Errorf("IsVideo(%s) should be false for file without extension", filePath)
		}
		if ext.IsSupported(filePath) {
			t.Errorf("IsSupported(%s) should be false for file without extension", filePath)
		}
		if ext.IsJPEG(filePath) {
			t.Errorf("IsJPEG(%s) should be false for file without extension", filePath)
		}
	}
}

func TestExtensions_IsVideo_AllFormats(t *testing.T) {
	ext := NewExtensions()

	videoFormats := []string{
		// Existing formats
		"test.mov", "test.MOV",
		"test.mp4", "test.MP4",
		// Common formats
		"test.avi", "test.AVI",
		"test.mkv", "test.MKV",
		"test.webm", "test.WEBM",
		"test.flv", "test.FLV",
		"test.wmv", "test.WMV",
		"test.m4v", "test.M4V",
		// Additional formats
		"test.3gp", "test.3GP",
		"test.m2ts", "test.M2TS",
		"test.mts", "test.MTS",
		"test.ogv", "test.OGV",
		"test.ts", "test.TS",
	}

	for _, format := range videoFormats {
		if !ext.IsVideo(format) {
			t.Errorf("Expected %s to be recognised as video", format)
		}
		if !ext.IsSupported(format) {
			t.Errorf("Expected %s to be recognised as supported", format)
		}
		if ext.IsImage(format) {
			t.Errorf("Expected %s to NOT be recognised as image", format)
		}
	}
}
