package pics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Helper functions

func createTestFileWithTime(t *testing.T, dir, filename string, modTime time.Time) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}
	return filePath
}

func assertTimeEqual(t *testing.T, expected, actual time.Time) {
	t.Helper()
	// Compare with truncation to second precision (file systems may not preserve nanoseconds)
	if !actual.Truncate(time.Second).Equal(expected.Truncate(time.Second)) {
		t.Errorf("Expected time %v, got %v", expected, actual)
	}
}

func TestModTimeExtractor_GetFileDate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with specific modification time
	testTime := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	testFile := createTestFileWithTime(t, tmpDir, "test.txt", testTime)

	// Test the extractor
	extractor := newModTimeExtractor()
	result, err := extractor.getFileDate(testFile)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the modification time is correct
	assertTimeEqual(t, testTime, result)
}

func TestModTimeExtractor_GetFileDate_NonexistentFile(t *testing.T) {
	extractor := newModTimeExtractor()
	_, err := extractor.getFileDate("/nonexistent/file.txt")

	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestModTimeExtractor_Name(t *testing.T) {
	extractor := newModTimeExtractor()
	if extractor.name() != "ModTime" {
		t.Errorf("Expected name 'ModTime', got '%s'", extractor.name())
	}
}

func TestExifDateExtractor_Name(t *testing.T) {
	extractor := newExifDateExtractor(createTestExiftool(t))
	if extractor.name() != "EXIF" {
		t.Errorf("Expected name 'EXIF', got '%s'", extractor.name())
	}
}

func TestAggregatedFileDateExtractor_FallbackToModTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file without EXIF data
	testTime := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	testFile := createTestFileWithTime(t, tmpDir, "test.txt", testTime)

	// Test the aggregated extractor
	extractor := NewFileDateExtractor(createTestExiftool(t))
	result, err := extractor.GetFileDate(testFile)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should fall back to ModTime since there's no EXIF data
	assertTimeEqual(t, testTime, result)
}

func TestAggregatedFileDateExtractor_NonexistentFile(t *testing.T) {
	extractor := NewFileDateExtractor(createTestExiftool(t))
	_, err := extractor.GetFileDate("/nonexistent/file.txt")

	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestAggregatedFileDateExtractor_ExtractorOrder(t *testing.T) {
	// Create a mock extractor that always succeeds
	successExtractor := &mockExtractor{
		returnDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		returnErr:  nil,
		nameStr:    "Success",
	}

	// Create a mock extractor that always fails
	failExtractor := &mockExtractor{
		returnErr: os.ErrNotExist,
		nameStr:   "Fail",
	}

	// Test with fail extractor first, success extractor second
	extractor := &AggregatedFileDateExtractor{
		extractors: []fileDateExtractor{
			failExtractor,
			successExtractor,
		},
	}

	result, err := extractor.GetFileDate("dummy.txt")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expectedDate) {
		t.Errorf("Expected date %v, got %v", expectedDate, result)
	}
}

func TestAggregatedFileDateExtractor_AllExtractorsFail(t *testing.T) {
	// Create mock extractors that all fail
	failExtractor1 := &mockExtractor{
		returnErr: os.ErrNotExist,
		nameStr:   "Fail1",
	}

	failExtractor2 := &mockExtractor{
		returnErr: os.ErrPermission,
		nameStr:   "Fail2",
	}

	extractor := &AggregatedFileDateExtractor{
		extractors: []fileDateExtractor{
			failExtractor1,
			failExtractor2,
		},
	}

	_, err := extractor.GetFileDate("dummy.txt")

	if err == nil {
		t.Error("Expected error when all extractors fail, got nil")
	}
}

// mockExtractor is a mock implementation for testing
type mockExtractor struct {
	returnDate time.Time
	returnErr  error
	nameStr    string
}

func (m *mockExtractor) getFileDate(filePath string) (time.Time, error) {
	if m.returnErr != nil {
		return time.Time{}, m.returnErr
	}
	return m.returnDate, nil
}

func (m *mockExtractor) name() string {
	return m.nameStr
}
