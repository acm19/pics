package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Helper functions

func createTempTestFile(t *testing.T, dir, filename string) {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
}

func TestCreateTempDir(t *testing.T) {
	tmpDir, cleanup, err := createTempDir(tempDirPrefix)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tmpDir == "" {
		t.Error("Expected non-empty temp directory path")
	}

	// Check directory exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Errorf("Expected directory to exist at %s", tmpDir)
	}

	// Test cleanup
	cleanup()

	// Check directory was removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Errorf("Expected directory to be removed at %s", tmpDir)
	}
}

func TestRunWorkerPool(t *testing.T) {
	jobs := []int{1, 2, 3, 4, 5}
	results := make([]int, 0)

	err := runWorkerPool(jobs, 2, func(job int) error {
		results = append(results, job*2)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(results) != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), len(results))
	}

	for i, job := range jobs {
		if results[i] != job*2 {
			t.Errorf("Expected result %d for job %d, got %d", job*2, job, results[i])
		}
	}
}

func TestRunWorkerPool_WithErrors(t *testing.T) {
	jobs := []int{1, 2, 3}

	err := runWorkerPool(jobs, 2, func(job int) error {
		if job == 2 {
			return nil
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error when all jobs succeed, got: %v", err)
	}
}

func TestRunWorkerPool_EmptyJobs(t *testing.T) {
	jobs := []int{}

	err := runWorkerPool(jobs, 2, func(job int) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error for empty jobs, got: %v", err)
	}
}

func TestS3Backup_ExtractETag(t *testing.T) {
	backup := &s3Backup{}

	tests := []struct {
		name     string
		etag     *string
		expected string
	}{
		{
			name:     "nil etag",
			etag:     nil,
			expected: "",
		},
		{
			name:     "empty etag",
			etag:     stringPtr(""),
			expected: "",
		},
		{
			name:     "etag with quotes",
			etag:     stringPtr(`"abc123"`),
			expected: "abc123",
		},
		{
			name:     "etag without quotes",
			etag:     stringPtr("abc123"),
			expected: "abc123",
		},
		{
			name:     "etag with single character",
			etag:     stringPtr(`"a"`),
			expected: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backup.extractETag(tt.etag)
			if result != tt.expected {
				t.Errorf("extractETag() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestS3Backup_CountMediaFiles(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		videoFiles     []string
		expectedImages int
		expectedVideos int
	}{
		{
			name: "with images and videos",
			files: []string{
				"image1.jpg",
				"image2.jpeg",
				"photo.heic",
				"document.txt", // Should be ignored
			},
			videoFiles: []string{
				"video1.mov",
				"video2.mp4",
			},
			expectedImages: 3,
			expectedVideos: 2,
		},
		{
			name: "only images no videos directory",
			files: []string{
				"image1.jpg",
				"image2.jpg",
			},
			videoFiles:     []string{},
			expectedImages: 2,
			expectedVideos: 0,
		},
		{
			name:           "empty directory",
			files:          []string{},
			videoFiles:     []string{},
			expectedImages: 0,
			expectedVideos: 0,
		},
		{
			name: "videos directory with no videos",
			files: []string{
				"image1.jpg",
			},
			videoFiles:     []string{},
			expectedImages: 1,
			expectedVideos: 0,
		},
		{
			name: "mixed supported and unsupported files",
			files: []string{
				"photo.jpg",
				"doc.pdf",
				"text.txt",
				"image.heic",
			},
			videoFiles: []string{
				"clip.mov",
				"video.avi", // Unsupported
			},
			expectedImages: 2,
			expectedVideos: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create files in root directory
			for _, filename := range tt.files {
				createTempTestFile(t, tmpDir, filename)
			}

			// Create videos subdirectory if there are video files
			if len(tt.videoFiles) > 0 {
				videosDir := filepath.Join(tmpDir, "videos")
				if err := os.MkdirAll(videosDir, 0755); err != nil {
					t.Fatalf("Failed to create videos directory: %v", err)
				}
				for _, filename := range tt.videoFiles {
					createTempTestFile(t, videosDir, filename)
				}
			}

			backup := &s3Backup{
				extensions: NewExtensions(),
			}

			images, videos, err := backup.countMediaFiles(tmpDir)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if images != tt.expectedImages {
				t.Errorf("Expected %d images, got %d", tt.expectedImages, images)
			}

			if videos != tt.expectedVideos {
				t.Errorf("Expected %d videos, got %d", tt.expectedVideos, videos)
			}
		})
	}
}

func TestS3Backup_MatchesFilter(t *testing.T) {
	backup := &s3Backup{}

	tests := []struct {
		name     string
		key      string
		filter   RestoreFilter
		expected bool
	}{
		{
			name:     "no filter",
			key:      "2023 06 June 15 vacation (10 images, 5 videos).tar.gz",
			filter:   RestoreFilter{},
			expected: true,
		},
		{
			name:   "matches from year",
			key:    "2023 06 June 15 vacation (10 images, 5 videos).tar.gz",
			filter: RestoreFilter{FromYear: 2023},
			expected: true,
		},
		{
			name:   "before from year",
			key:    "2022 06 June 15 vacation (10 images, 5 videos).tar.gz",
			filter: RestoreFilter{FromYear: 2023},
			expected: false,
		},
		{
			name:   "matches to year",
			key:    "2023 06 June 15 vacation (10 images, 5 videos).tar.gz",
			filter: RestoreFilter{ToYear: 2023},
			expected: true,
		},
		{
			name:   "after to year",
			key:    "2024 06 June 15 vacation (10 images, 5 videos).tar.gz",
			filter: RestoreFilter{ToYear: 2023},
			expected: false,
		},
		{
			name:   "matches year and month range",
			key:    "2023 06 June 15 vacation (10 images, 5 videos).tar.gz",
			filter: RestoreFilter{FromYear: 2023, FromMonth: 1, ToYear: 2023, ToMonth: 12},
			expected: true,
		},
		{
			name:   "before from month",
			key:    "2023 05 May 15 vacation (10 images, 5 videos).tar.gz",
			filter: RestoreFilter{FromYear: 2023, FromMonth: 6},
			expected: false,
		},
		{
			name:   "after to month",
			key:    "2023 07 July 15 vacation (10 images, 5 videos).tar.gz",
			filter: RestoreFilter{ToYear: 2023, ToMonth: 6},
			expected: false,
		},
		{
			name:     "invalid key format",
			key:      "invalid",
			filter:   RestoreFilter{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backup.matchesFilter(tt.key, tt.filter)
			if result != tt.expected {
				t.Errorf("matchesFilter() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestS3Backup_ExtractDirNameFromKey(t *testing.T) {
	backup := &s3Backup{}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "with counts suffix",
			key:      "2023 06 June 15 vacation (10 images, 5 videos).tar.gz",
			expected: "2023 06 June 15 vacation",
		},
		{
			name:     "without counts suffix",
			key:      "2023 06 June 15 vacation.tar.gz",
			expected: "2023 06 June 15 vacation",
		},
		{
			name:     "simple name",
			key:      "vacation.tar.gz",
			expected: "vacation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backup.extractDirNameFromKey(tt.key)
			if result != tt.expected {
				t.Errorf("extractDirNameFromKey() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "NotFound type",
			err:      &types.NotFound{Message: stringPtr("not found")},
			expected: true,
		},
		{
			name:     "generic error",
			err:      os.ErrNotExist,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("isNotFoundError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestS3Backup_CalculateMD5(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with known content
	filePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	backup := &s3Backup{}
	hash, err := backup.calculateMD5(filePath)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	// Hash should be consistent
	hash2, err := backup.calculateMD5(filePath)
	if err != nil {
		t.Errorf("Expected no error on second call, got: %v", err)
	}

	if hash != hash2 {
		t.Errorf("Expected consistent hash, got %s and %s", hash, hash2)
	}
}

func TestS3Backup_CalculateMD5_NonexistentFile(t *testing.T) {
	backup := &s3Backup{}
	_, err := backup.calculateMD5("/nonexistent/file.txt")

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// Mock API error for testing
type mockAPIError struct {
	code string
}

func (m *mockAPIError) Error() string {
	return m.code
}

func (m *mockAPIError) ErrorCode() string {
	return m.code
}

func (m *mockAPIError) ErrorMessage() string {
	return m.code
}

func (m *mockAPIError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultUnknown
}

func TestIsNotFoundError_APIError(t *testing.T) {
	err := &mockAPIError{code: "NotFound"}
	if !isNotFoundError(err) {
		t.Error("Expected isNotFoundError to return true for NotFound API error")
	}

	err2 := &mockAPIError{code: "OtherError"}
	if isNotFoundError(err2) {
		t.Error("Expected isNotFoundError to return false for other API error")
	}
}
