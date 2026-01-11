package pics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileRenamer_RenameFilesWithPattern(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test files
	createFile(t, testDir, "image1.jpg")
	createFile(t, testDir, "image2.JPG")
	createFile(t, testDir, "image3.JPEG")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	count, err := renamer.RenameFilesWithPattern(testDir, "test_prefix", ext.IsImage, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 files renamed, got: %d", count)
	}

	// Check files were renamed with normalised extensions
	assertFileExists(t, filepath.Join(testDir, "test_prefix_00001.jpg"))
	assertFileExists(t, filepath.Join(testDir, "test_prefix_00002.jpg"))
	assertFileExists(t, filepath.Join(testDir, "test_prefix_00003.jpeg"))

	// Check original files were removed
	assertFileNotExists(t, filepath.Join(testDir, "image1.jpg"))
	assertFileNotExists(t, filepath.Join(testDir, "image2.JPG"))
	assertFileNotExists(t, filepath.Join(testDir, "image3.JPEG"))
}

func TestFileRenamer_RenameFilesWithPattern_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	count, err := renamer.RenameFilesWithPattern(testDir, "test_prefix", ext.IsImage, nil)

	if err != nil {
		t.Errorf("Expected no error for empty directory, got: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 files renamed for empty directory, got: %d", count)
	}
}

func TestFileRenamer_RenameFilesWithPattern_NoMatchingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create non-image files
	createFile(t, testDir, "document.txt")
	createFile(t, testDir, "data.csv")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.RenameFilesWithPattern(testDir, "test_prefix", ext.IsImage, nil)

	if err != nil {
		t.Errorf("Expected no error when no matching files, got: %v", err)
	}

	// Check original files still exist
	assertFileExists(t, filepath.Join(testDir, "document.txt"))
	assertFileExists(t, filepath.Join(testDir, "data.csv"))
}

func TestFileRenamer_RenameFilesWithPattern_SkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create subdirectory and files
	subDir := filepath.Join(testDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	createFile(t, testDir, "image1.jpg")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.RenameFilesWithPattern(testDir, "test_prefix", ext.IsImage, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check file was renamed
	assertFileExists(t, filepath.Join(testDir, "test_prefix_00001.jpg"))

	// Check subdirectory still exists
	assertFileExists(t, subDir)
}

func TestFileRenamer_RenameFilesWithPattern_SortedOrder(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create files in non-alphabetical order
	createFile(t, testDir, "z.jpg")
	createFile(t, testDir, "a.jpg")
	createFile(t, testDir, "m.jpg")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.RenameFilesWithPattern(testDir, "sorted", ext.IsImage, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Files should be renamed in sorted order: a.jpg, m.jpg, z.jpg
	assertFileExists(t, filepath.Join(testDir, "sorted_00001.jpg"))
	assertFileExists(t, filepath.Join(testDir, "sorted_00002.jpg"))
	assertFileExists(t, filepath.Join(testDir, "sorted_00003.jpg"))
}

func TestFileRenamer_RenameFilesWithPattern_NormalisesExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create files with uppercase extensions
	createFile(t, testDir, "image1.JPG")
	createFile(t, testDir, "image2.HEIC")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.RenameFilesWithPattern(testDir, "normalised", ext.IsImage, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check files have lowercase extensions
	assertFileExists(t, filepath.Join(testDir, "normalised_00001.jpg"))
	assertFileExists(t, filepath.Join(testDir, "normalised_00002.heic"))
}

func TestFileRenamer_MoveAndRenameFilesWithPattern(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create test files
	createFile(t, sourceDir, "video1.mov")
	createFile(t, sourceDir, "video2.MOV")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	count, err := renamer.MoveAndRenameFilesWithPattern(sourceDir, targetDir, "vid_prefix", ext.IsVideo, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files moved and renamed, got: %d", count)
	}

	// Check target directory was created
	assertFileExists(t, targetDir)

	// Check files were moved and renamed with lowercase extensions
	assertFileExists(t, filepath.Join(targetDir, "vid_prefix_00001.mov"))
	assertFileExists(t, filepath.Join(targetDir, "vid_prefix_00002.mov"))

	// Check files were removed from source
	assertFileNotExists(t, filepath.Join(sourceDir, "video1.mov"))
	assertFileNotExists(t, filepath.Join(sourceDir, "video2.MOV"))
}

func TestFileRenamer_MoveAndRenameFilesWithPattern_NormalisesExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create test files with uppercase extensions
	createFile(t, sourceDir, "video1.MP4")
	createFile(t, sourceDir, "video2.MOV")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.MoveAndRenameFilesWithPattern(sourceDir, targetDir, "video", ext.IsVideo, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check files have lowercase extensions
	assertFileExists(t, filepath.Join(targetDir, "video_00001.mp4"))
	assertFileExists(t, filepath.Join(targetDir, "video_00002.mov"))
}

func TestFileRenamer_MoveAndRenameFilesWithPattern_EmptySource(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.MoveAndRenameFilesWithPattern(sourceDir, targetDir, "prefix", ext.IsVideo, nil)

	if err != nil {
		t.Errorf("Expected no error for empty source, got: %v", err)
	}

	// Target directory should not be created when there are no files
	assertFileNotExists(t, targetDir)
}

func TestFileRenamer_MoveAndRenameFilesWithPattern_NoMatchingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create non-video files
	createFile(t, sourceDir, "document.txt")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.MoveAndRenameFilesWithPattern(sourceDir, targetDir, "prefix", ext.IsVideo, nil)

	if err != nil {
		t.Errorf("Expected no error when no matching files, got: %v", err)
	}

	// Target directory should not be created
	assertFileNotExists(t, targetDir)

	// Original file should still exist
	assertFileExists(t, filepath.Join(sourceDir, "document.txt"))
}

func TestFileRenamer_MoveAndRenameFilesWithPattern_InPlace(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")

	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test files
	createFile(t, testDir, "vid1.mov")
	createFile(t, testDir, "vid2.MOV")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	// Move to same directory (rename in place)
	_, err := renamer.MoveAndRenameFilesWithPattern(testDir, testDir, "video", ext.IsVideo, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check files were renamed in place with lowercase extensions
	assertFileExists(t, filepath.Join(testDir, "video_00001.mov"))
	assertFileExists(t, filepath.Join(testDir, "video_00002.mov"))
}

func TestFileRenamer_MoveAndRenameFilesWithPattern_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "target")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.MoveAndRenameFilesWithPattern("/nonexistent/source", targetDir, "prefix", ext.IsImage, nil)

	if err == nil {
		t.Error("Expected error for nonexistent source directory")
	}
}

func TestFileRenamer_MoveAndRenameFilesWithPattern_CreatesTargetOnlyWhenNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create mixed files
	createFile(t, sourceDir, "video.mov")
	createFile(t, sourceDir, "document.txt")

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	_, err := renamer.MoveAndRenameFilesWithPattern(sourceDir, targetDir, "vid", ext.IsVideo, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Target directory should be created because there's a matching video
	assertFileExists(t, targetDir)
	assertFileExists(t, filepath.Join(targetDir, "vid_00001.mov"))

	// Non-matching file should remain in source
	assertFileExists(t, filepath.Join(sourceDir, "document.txt"))
}

func TestFileRenamer_SortByDateThenFilename(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create valid JPEG files with same date but different names
	sameDate := parseTime(t, "2023-06-15T10:00:00Z")
	createValidJPEGWithDate(t, testDir, "photo_c.jpg", sameDate)
	createValidJPEGWithDate(t, testDir, "photo_a.jpg", sameDate)
	createValidJPEGWithDate(t, testDir, "photo_b.jpg", sameDate)

	renamer := NewFileRenamer(createTestExiftool(t))
	ext := NewExtensions()
	count, err := renamer.RenameFilesWithPattern(testDir, "sorted", ext.IsImage, nil)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 files renamed, got: %d", count)
	}

	// Files with same date should be sorted alphabetically by filename
	assertFileExists(t, filepath.Join(testDir, "sorted_00001.jpg")) // photo_a
	assertFileExists(t, filepath.Join(testDir, "sorted_00002.jpg")) // photo_b
	assertFileExists(t, filepath.Join(testDir, "sorted_00003.jpg")) // photo_c
}

func parseTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		t.Fatalf("Failed to parse time %s: %v", timeStr, err)
	}
	return parsed
}
