package pics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test-level options used across all tests
var testParseOptions = ParseOptions{
	CompressJPEGs:  false, // Disable compression since test files aren't real JPEGs
	JPEGQuality:    50,
	TempDirName:    "tmp_image",
	MaxConcurrency: 100,
}

func createTestParser(t *testing.T) MediaParser {
	t.Helper()
	organiser := NewFileOrganiser(createTestExiftool(t))
	return NewMediaParser("", organiser)
}

// Helper functions

func createSourceAndTarget(t *testing.T, tmpDir string) (string, string) {
	t.Helper()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}
	return sourceDir, targetDir
}

func createMediaFile(t *testing.T, dir, filename string, modTime time.Time) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("test media content"), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", filename, err)
	}
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}
	return filePath
}

func createSubdir(t *testing.T, parentDir, name string) string {
	t.Helper()
	subdirPath := filepath.Join(parentDir, name)
	if err := os.MkdirAll(subdirPath, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	return subdirPath
}

func assertMediaFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Expected file to exist at %s", path)
	}
}

func assertMediaFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Expected file to not exist at %s", path)
	}
}

func assertFileModTime(t *testing.T, path string, expectedModTime time.Time) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat file %s: %v", path, err)
	}
	// Compare modification times with 1 second tolerance
	actualModTime := info.ModTime()
	if actualModTime.Sub(expectedModTime).Abs() > time.Second {
		t.Errorf("Expected mod time %v, got %v", expectedModTime, actualModTime)
	}
}

func TestMediaParser_Parse(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create test files with specific modification times
	testDate := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	createMediaFile(t, sourceDir, "image1.jpg", testDate)
	createMediaFile(t, sourceDir, "image2.jpeg", testDate)
	createMediaFile(t, sourceDir, "video1.mov", testDate)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that files were organised into date-based directory
	expectedDir := filepath.Join(targetDir, "2023 06 June 15")
	assertMediaFileExists(t, expectedDir)

	// Check images were renamed
	assertMediaFileExists(t, filepath.Join(expectedDir, "2023_06_June_15_00001.jpg"))
	assertMediaFileExists(t, filepath.Join(expectedDir, "2023_06_June_15_00002.jpeg"))

	// Check videos were moved to subdirectory and renamed
	videosDir := filepath.Join(expectedDir, "videos")
	assertMediaFileExists(t, videosDir)
	assertMediaFileExists(t, filepath.Join(videosDir, "2023_06_June_15_00001.mov"))
}

func TestMediaParser_Parse_EmptySource(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Parse with no files in source
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error for empty source, got: %v", err)
	}
}

func TestMediaParser_Parse_MultipleDates(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create files with different dates
	date1 := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	date2 := time.Date(2023, 7, 20, 14, 0, 0, 0, time.UTC)

	createMediaFile(t, sourceDir, "june.jpg", date1)
	createMediaFile(t, sourceDir, "july.jpg", date2)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check both date directories were created
	assertMediaFileExists(t, filepath.Join(targetDir, "2023 06 June 15", "2023_06_June_15_00001.jpg"))
	assertMediaFileExists(t, filepath.Join(targetDir, "2023 07 July 20", "2023_07_July_20_00001.jpg"))
}

func TestMediaParser_Parse_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create files in subdirectories
	testDate := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	subdir1 := createSubdir(t, sourceDir, "folder1")
	subdir2 := createSubdir(t, sourceDir, "folder2")

	createMediaFile(t, subdir1, "image1.jpg", testDate)
	createMediaFile(t, subdir2, "image2.jpeg", testDate)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that files from subdirectories were processed
	expectedDir := filepath.Join(targetDir, "2023 06 June 15")
	assertMediaFileExists(t, expectedDir)

	// Files should exist with prefixed names
	entries, err := os.ReadDir(expectedDir)
	if err != nil {
		t.Fatalf("Failed to read target directory: %v", err)
	}

	// Should have 2 image files
	imageCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".jpg" || filepath.Ext(entry.Name()) == ".jpeg") {
			imageCount++
		}
	}

	if imageCount != 2 {
		t.Errorf("Expected 2 images, got %d", imageCount)
	}
}

func TestMediaParser_Parse_SkipsDotFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create regular file and dot file
	testDate := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	createMediaFile(t, sourceDir, "image.jpg", testDate)
	createMediaFile(t, sourceDir, ".hidden.jpg", testDate)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check only 1 file was processed (dot file should be skipped)
	expectedDir := filepath.Join(targetDir, "2023 06 June 15")
	entries, err := os.ReadDir(expectedDir)
	if err != nil {
		t.Fatalf("Failed to read target directory: %v", err)
	}

	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			fileCount++
		}
	}

	if fileCount != 1 {
		t.Errorf("Expected 1 file (dot file should be skipped), got %d", fileCount)
	}
}

func TestMediaParser_Parse_SkipsDotDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create file in regular directory and dot directory
	testDate := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	regularSubdir := createSubdir(t, sourceDir, "photos")
	dotSubdir := createSubdir(t, sourceDir, ".hidden")

	createMediaFile(t, regularSubdir, "image1.jpg", testDate)
	createMediaFile(t, dotSubdir, "image2.jpg", testDate)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check only 1 file was processed (file in dot directory should be skipped)
	expectedDir := filepath.Join(targetDir, "2023 06 June 15")
	entries, err := os.ReadDir(expectedDir)
	if err != nil {
		t.Fatalf("Failed to read target directory: %v", err)
	}

	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			fileCount++
		}
	}

	if fileCount != 1 {
		t.Errorf("Expected 1 file (files in dot directories should be skipped), got %d", fileCount)
	}
}

func TestMediaParser_Parse_MixedFileTypes(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create various media file types
	testDate := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	createMediaFile(t, sourceDir, "image.jpg", testDate)
	createMediaFile(t, sourceDir, "photo.jpeg", testDate)
	createMediaFile(t, sourceDir, "picture.heic", testDate)
	createMediaFile(t, sourceDir, "video.mov", testDate)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check all file types were processed (sorted alphabetically: image.jpg, photo.jpeg, picture.heic)
	expectedDir := filepath.Join(targetDir, "2023 06 June 15")
	assertMediaFileExists(t, filepath.Join(expectedDir, "2023_06_June_15_00001.jpg"))
	assertMediaFileExists(t, filepath.Join(expectedDir, "2023_06_June_15_00002.jpeg"))
	assertMediaFileExists(t, filepath.Join(expectedDir, "2023_06_June_15_00003.heic"))

	videosDir := filepath.Join(expectedDir, "videos")
	assertMediaFileExists(t, filepath.Join(videosDir, "2023_06_June_15_00001.mov"))
}

func TestMediaParser_Parse_IgnoresUnsupportedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create supported and unsupported files
	testDate := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	createMediaFile(t, sourceDir, "image.jpg", testDate)
	createMediaFile(t, sourceDir, "document.txt", testDate)
	createMediaFile(t, sourceDir, "video.avi", testDate)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check only supported file was processed
	expectedDir := filepath.Join(targetDir, "2023 06 June 15")
	assertMediaFileExists(t, filepath.Join(expectedDir, "2023_06_June_15_00001.jpg"))

	// Unsupported files should not be in target
	assertMediaFileNotExists(t, filepath.Join(expectedDir, "document.txt"))
	assertMediaFileNotExists(t, filepath.Join(expectedDir, "video.avi"))
}

func TestMediaParser_Parse_MP4Videos(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create MP4 videos and images
	testDate := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	createMediaFile(t, sourceDir, "image.jpg", testDate)
	createMediaFile(t, sourceDir, "video1.mp4", testDate)
	createMediaFile(t, sourceDir, "video2.MP4", testDate)

	// Parse files
	err := createTestParser(t).Parse(sourceDir, targetDir, testParseOptions)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check image was processed
	expectedDir := filepath.Join(targetDir, "2023 06 June 15")
	assertMediaFileExists(t, filepath.Join(expectedDir, "2023_06_June_15_00001.jpg"))

	// Check MP4 videos were moved to subdirectory and renamed with lowercase extensions
	videosDir := filepath.Join(expectedDir, "videos")
	assertMediaFileExists(t, videosDir)
	assertMediaFileExists(t, filepath.Join(videosDir, "2023_06_June_15_00001.mp4"))
	assertMediaFileExists(t, filepath.Join(videosDir, "2023_06_June_15_00002.mp4"))
}

func TestCopyFilePreserveTime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file with specific mod time
	srcPath := filepath.Join(tmpDir, "source.txt")
	modTime := time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC)
	if err := os.WriteFile(srcPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	if err := os.Chtimes(srcPath, modTime, modTime); err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "destination.txt")
	err := copyFilePreserveTime(srcPath, dstPath)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check destination file exists
	assertMediaFileExists(t, dstPath)

	// Check content was copied
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got %s", string(content))
	}

	// Check modification time was preserved
	assertFileModTime(t, dstPath, modTime)
}

func TestCopyFilePreserveTime_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "destination.txt")

	err := copyFilePreserveTime(srcPath, dstPath)

	if err == nil {
		t.Error("Expected error for nonexistent source file")
	}
}

func TestCopyFilePreserveTime_InvalidDestination(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Try to copy to invalid destination (directory that doesn't exist)
	dstPath := filepath.Join(tmpDir, "nonexistent_dir", "destination.txt")

	err := copyFilePreserveTime(srcPath, dstPath)

	if err == nil {
		t.Error("Expected error for invalid destination path")
	}
}

func TestDefaultParseOptions(t *testing.T) {
	opts := DefaultParseOptions()

	// Check default values
	if !opts.CompressJPEGs {
		t.Error("Expected CompressJPEGs to be true by default")
	}

	if opts.JPEGQuality != 50 {
		t.Errorf("Expected JPEGQuality to be 50, got %d", opts.JPEGQuality)
	}

	if opts.TempDirName != "tmp_image" {
		t.Errorf("Expected TempDirName to be 'tmp_image', got %s", opts.TempDirName)
	}

	if opts.MaxConcurrency != 100 {
		t.Errorf("Expected MaxConcurrency to be 100, got %d", opts.MaxConcurrency)
	}
}

func TestMediaParser_ParseWithProgressChannel(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir, targetDir := createSourceAndTarget(t, tmpDir)

	// Create test files
	subdir := filepath.Join(sourceDir, "vacation")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	testFiles := []string{"photo1.jpg", "photo2.jpg", "video.mov"}
	for _, filename := range testFiles {
		filePath := filepath.Join(subdir, filename)
		if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create progress channel with buffer
	progressChan := make(chan ProgressEvent, 100)

	// Run parse with progress channel
	opts := testParseOptions
	opts.ProgressChan = progressChan

	// Run parse in goroutine so we can read from channel
	done := make(chan error)
	go func() {
		done <- createTestParser(t).Parse(sourceDir, targetDir, opts)
	}()

	// Collect progress events
	var events []ProgressEvent
	timeout := time.After(5 * time.Second)
	isDone := false

	for !isDone {
		select {
		case event := <-progressChan:
			events = append(events, event)
		case err := <-done:
			// Parse finished, drain any remaining events
			close(progressChan)
			for event := range progressChan {
				events = append(events, event)
			}
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			isDone = true
		case <-timeout:
			t.Fatal("Test timed out waiting for parse to complete")
		}
	}

	// Verify we received progress events
	if len(events) == 0 {
		t.Fatal("Expected to receive progress events, got none")
	}

	t.Logf("Received %d progress events", len(events))

	// Verify event structure
	stages := make(map[string]int)

	for i, event := range events {
		stages[event.Stage]++

		// Verify required fields are set
		if event.Stage == "" {
			t.Errorf("Event %d: missing Stage", i)
		}
		if event.Message == "" {
			t.Errorf("Event %d: missing Message", i)
		}
		if event.Current < 0 {
			t.Errorf("Event %d: invalid Current value: %d", i, event.Current)
		}
		if event.Total < 0 {
			t.Errorf("Event %d: invalid Total value: %d", i, event.Total)
		}
		if event.Current > event.Total {
			t.Errorf("Event %d: Current (%d) exceeds Total (%d)", i, event.Current, event.Total)
		}
		if event.Total > 0 && event.Current == 0 {
			t.Errorf("Event %d: Current is 0 but Total is %d", i, event.Total)
		}
	}

	// Verify we got expected stages
	expectedStages := []string{"copying", "organising", "renaming"}
	for _, stage := range expectedStages {
		if count, found := stages[stage]; !found {
			t.Errorf("Expected to see stage '%s' but it was not present", stage)
		} else {
			t.Logf("Stage '%s': %d events", stage, count)
		}
	}

	// Print sample events for debugging
	if len(events) > 0 {
		t.Logf("Sample event: Stage=%s, Current=%d, Total=%d, Message=%s",
			events[0].Stage, events[0].Current, events[0].Total, events[0].Message)
	}
}
