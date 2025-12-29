package pics

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper functions

func createTestDirectory(t *testing.T, parentDir, dirName string) string {
	t.Helper()
	dirPath := filepath.Join(parentDir, dirName)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dirPath, err)
	}
	return dirPath
}

func createTestImage(t *testing.T, dir, filename string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("test image"), 0644); err != nil {
		t.Fatalf("Failed to create image %s: %v", filePath, err)
	}
	return filePath
}

func createTestVideo(t *testing.T, dir, filename string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("test video"), 0644); err != nil {
		t.Fatalf("Failed to create video %s: %v", filePath, err)
	}
	return filePath
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Expected directory to exist at %s", path)
	}
}

func assertFilesExist(t *testing.T, dir string, filenames []string) {
	t.Helper()
	for _, filename := range filenames {
		filePath := filepath.Join(dir, filename)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("Expected file %s to exist", filename)
		}
	}
}

func TestNewDirectoryRenamer(t *testing.T) {
	renamer := NewDirectoryRenamer()
	if renamer == nil {
		t.Error("Expected non-nil renamer")
	}
}

func TestDirectoryRenamer_RenameDirectory_WithImages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test directory with the expected format
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15")

	// Create test images
	createTestImage(t, testDir, "img1.jpg")
	createTestImage(t, testDir, "img2.JPG")
	createTestImage(t, testDir, "img3.jpeg")

	// Rename directory
	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "vacation")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that directory was renamed
	newDirPath := filepath.Join(tmpDir, "2023 06 June 15 vacation")
	assertDirExists(t, newDirPath)

	// Check that images were renamed
	expectedFiles := []string{
		"2023_06_June_15_vacation_00001.jpg",
		"2023_06_June_15_vacation_00002.jpg",
		"2023_06_June_15_vacation_00003.jpeg",
	}
	assertFilesExist(t, newDirPath, expectedFiles)
}

func TestDirectoryRenamer_RenameDirectory_WithVideos(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test directory with videos subdirectory
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15")
	videosDir := createTestDirectory(t, testDir, "videos")

	// Create test videos
	createTestVideo(t, videosDir, "vid1.mov")
	createTestVideo(t, videosDir, "vid2.MOV")

	// Rename directory
	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "trip")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that directory was renamed
	newDirPath := filepath.Join(tmpDir, "2023 06 June 15 trip")
	assertDirExists(t, newDirPath)

	// Check that videos were renamed with lowercase extensions
	newVideosDir := filepath.Join(newDirPath, "videos")
	expectedFiles := []string{
		"2023_06_June_15_trip_00001.mov",
		"2023_06_June_15_trip_00002.mov",
	}
	assertFilesExist(t, newVideosDir, expectedFiles)
}

func TestDirectoryRenamer_RenameDirectory_WithImagesAndVideos(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test directory
	testDir := createTestDirectory(t, tmpDir, "2023 12 December 25")

	// Create test images
	createTestImage(t, testDir, "img1.jpg")
	createTestImage(t, testDir, "img2.heic")

	// Create videos subdirectory with videos
	videosDir := createTestDirectory(t, testDir, "videos")
	createTestVideo(t, videosDir, "vid1.mov")

	// Rename directory
	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "christmas")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that directory was renamed
	newDirPath := filepath.Join(tmpDir, "2023 12 December 25 christmas")
	assertDirExists(t, newDirPath)

	// Check images were renamed
	expectedImages := []string{
		"2023_12_December_25_christmas_00001.jpg",
		"2023_12_December_25_christmas_00002.heic",
	}
	assertFilesExist(t, newDirPath, expectedImages)

	// Check videos were renamed
	newVideosDir := filepath.Join(newDirPath, "videos")
	assertFilesExist(t, newVideosDir, []string{"2023_12_December_25_christmas_00001.mov"})
}

func TestDirectoryRenamer_RenameDirectory_EmptyName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test directory with existing name
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15 oldname")

	// Create test image
	createTestImage(t, testDir, "img1.jpg")

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that directory was renamed (name removed)
	newDirPath := filepath.Join(tmpDir, "2023 06 June 15")
	if _, err := os.Stat(newDirPath); err != nil {
		t.Errorf("Expected directory to exist at %s", newDirPath)
	}

	// Check that image was renamed
	expectedFile := filepath.Join(newDirPath, "2023_06_June_15_00001.jpg")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("Expected file to exist at %s", expectedFile)
	}
}

func TestDirectoryRenamer_RenameDirectory_NoChange(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test directory
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15 vacation")

	// Create test image
	createTestImage(t, testDir, "img1.jpg")

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "vacation")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Directory should still exist with same name
	if _, err := os.Stat(testDir); err != nil {
		t.Errorf("Expected directory to still exist at %s", testDir)
	}

	// Image should be renamed to match the pattern
	expectedFile := filepath.Join(testDir, "2023_06_June_15_vacation_00001.jpg")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("Expected file to exist at %s", expectedFile)
	}
}

func TestDirectoryRenamer_RenameDirectory_NonexistentDirectory(t *testing.T) {
	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory("/nonexistent/directory", "newname")

	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

func TestDirectoryRenamer_RenameDirectory_NotADirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file, not a directory
	filePath := filepath.Join(tmpDir, "notadir.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(filePath, "newname")

	if err == nil {
		t.Error("Expected error for file instead of directory, got nil")
	}
}

func TestDirectoryRenamer_RenameDirectory_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with invalid format (missing parts)
	testDir := createTestDirectory(t, tmpDir, "2023 06 June")

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "newname")

	if err == nil {
		t.Error("Expected error for invalid directory name format, got nil")
	}
}

func TestDirectoryRenamer_RenameDirectory_TargetExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source directory
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15")

	// Create target directory that will conflict
	createTestDirectory(t, tmpDir, "2023 06 June 15 vacation")

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "vacation")

	if err == nil {
		t.Error("Expected error when target directory already exists, got nil")
	}
}

func TestDirectoryRenamer_RenameDirectory_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty directory
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15")

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "empty")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that directory was renamed
	newDirPath := filepath.Join(tmpDir, "2023 06 June 15 empty")
	if _, err := os.Stat(newDirPath); err != nil {
		t.Errorf("Expected directory to exist at %s", newDirPath)
	}
}

func TestDirectoryRenamer_RenameDirectory_PreservesExtensionCase(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15")

	// Create images with different extension cases
	createTestImage(t, testDir, "img1.JPG")
	createTestImage(t, testDir, "img2.jpeg")
	createTestImage(t, testDir, "img3.HEIC")

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "test")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	newDirPath := filepath.Join(tmpDir, "2023 06 June 15 test")

	// Check that extensions are lowercased
	expectedFiles := []string{
		"2023_06_June_15_test_00001.jpg",
		"2023_06_June_15_test_00002.jpeg",
		"2023_06_June_15_test_00003.heic",
	}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(newDirPath, filename)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("Expected file %s to exist", filename)
		}
	}
}

func TestDirectoryRenamer_RenameDirectory_SortsFilesAlphabetically(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory
	testDir := createTestDirectory(t, tmpDir, "2023 06 June 15")

	// Create images in non-alphabetical order
	createTestImage(t, testDir, "zzz.jpg")
	createTestImage(t, testDir, "aaa.jpg")
	createTestImage(t, testDir, "mmm.jpg")

	renamer := NewDirectoryRenamer()
	err := renamer.RenameDirectory(testDir, "sorted")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	newDirPath := filepath.Join(tmpDir, "2023 06 June 15 sorted")

	// Files should be renamed in alphabetical order
	// aaa.jpg -> _00001.jpg, mmm.jpg -> _00002.jpg, zzz.jpg -> _00003.jpg
	expectedFiles := []string{
		"2023_06_June_15_sorted_00001.jpg",
		"2023_06_June_15_sorted_00002.jpg",
		"2023_06_June_15_sorted_00003.jpg",
	}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(newDirPath, filename)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("Expected file %s to exist", filename)
		}
	}
}
