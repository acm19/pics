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
		{"video.mov", false},
		{"video.mp4", false},
		{"document.txt", false},
		{"file.png", false},
		{"/path/to/image.jpg", true},
		{"/path/to/image.HEIC", true},
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
		{"file.avi", false},
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
		// Videos
		{"video.mov", true},
		{"video.mp4", true},
		{"video.MP4", true},
		// Unsupported
		{"document.txt", false},
		{"file.png", false},
		{"file.avi", false},
		{"file.gif", false},
		{"/path/to/supported.jpg", true},
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
