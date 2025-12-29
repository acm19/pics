package pics

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper functions

func createTestDir(t *testing.T, parentDir, name string) string {
	t.Helper()
	dirPath := filepath.Join(parentDir, name)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dirPath, err)
	}
	return dirPath
}

func createTestFile(t *testing.T, dir, filename string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", filePath, err)
	}
	return filePath
}

func TestFileStats_ValidateDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source and target directories
	sourceDir := createTestDir(t, tmpDir, "source")
	targetDir := createTestDir(t, tmpDir, "target")

	stats := NewFileStats()
	err := stats.ValidateDirectories(sourceDir, targetDir)

	if err != nil {
		t.Errorf("Expected no error for valid directories, got: %v", err)
	}
}

func TestFileStats_ValidateDirectories_NonexistentSource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only target directory
	targetDir := createTestDir(t, tmpDir, "target")
	nonexistentSource := filepath.Join(tmpDir, "nonexistent")

	stats := NewFileStats()
	err := stats.ValidateDirectories(nonexistentSource, targetDir)

	if err == nil {
		t.Error("Expected error for nonexistent source directory")
	}
}

func TestFileStats_ValidateDirectories_NonexistentTarget(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only source directory
	sourceDir := createTestDir(t, tmpDir, "source")
	nonexistentTarget := filepath.Join(tmpDir, "nonexistent")

	stats := NewFileStats()
	err := stats.ValidateDirectories(sourceDir, nonexistentTarget)

	if err == nil {
		t.Error("Expected error for nonexistent target directory")
	}
}

func TestFileStats_ValidateDirectories_SourceIsFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file instead of source directory
	sourceFile := createTestFile(t, tmpDir, "source.txt")
	targetDir := createTestDir(t, tmpDir, "target")

	stats := NewFileStats()
	err := stats.ValidateDirectories(sourceFile, targetDir)

	if err == nil {
		t.Error("Expected error when source is a file, not a directory")
	}
}

func TestFileStats_ValidateDirectories_TargetIsFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source directory and target file
	sourceDir := createTestDir(t, tmpDir, "source")
	targetFile := createTestFile(t, tmpDir, "target.txt")

	stats := NewFileStats()
	err := stats.ValidateDirectories(sourceDir, targetFile)

	if err == nil {
		t.Error("Expected error when target is a file, not a directory")
	}
}

func TestFileStats_GetFileCount(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files in root directory
	createTestFile(t, tmpDir, "file1.txt")
	createTestFile(t, tmpDir, "file2.jpg")
	createTestFile(t, tmpDir, "file3.mov")

	// Create subdirectory with files
	subDir := createTestDir(t, tmpDir, "subdir")
	createTestFile(t, subDir, "file4.txt")
	createTestFile(t, subDir, "file5.jpg")

	stats := NewFileStats()
	count, err := stats.GetFileCount(tmpDir)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should count all 5 files
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

func TestFileStats_GetFileCount_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	stats := NewFileStats()
	count, err := stats.GetFileCount(tmpDir)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should count 0 files in empty directory
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

func TestFileStats_GetFileCount_NonexistentDirectory(t *testing.T) {
	stats := NewFileStats()
	_, err := stats.GetFileCount("/nonexistent/directory")

	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

func TestFileStats_GetFileCount_SkipsDotFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create regular files
	createTestFile(t, tmpDir, "file1.txt")
	createTestFile(t, tmpDir, "file2.jpg")

	// Create dot files (should be skipped)
	createTestFile(t, tmpDir, ".hidden")
	createTestFile(t, tmpDir, ".DS_Store")

	// Create dot directory with files (should be skipped)
	dotDir := createTestDir(t, tmpDir, ".dotdir")
	createTestFile(t, dotDir, "file3.txt")

	stats := NewFileStats()
	count, err := stats.GetFileCount(tmpDir)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should count only 2 regular files, skipping dot files and dot directories
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestFileStats_GetFileCount_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files at multiple nesting levels
	createTestFile(t, tmpDir, "root1.txt")
	createTestFile(t, tmpDir, "root2.txt")

	level1 := createTestDir(t, tmpDir, "level1")
	createTestFile(t, level1, "file1.txt")

	level2 := createTestDir(t, level1, "level2")
	createTestFile(t, level2, "file2.txt")
	createTestFile(t, level2, "file3.txt")

	level3 := createTestDir(t, level2, "level3")
	createTestFile(t, level3, "file4.txt")

	stats := NewFileStats()
	count, err := stats.GetFileCount(tmpDir)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should count all 6 files across all levels
	if count != 6 {
		t.Errorf("Expected count 6, got %d", count)
	}
}

func TestFileStats_GetFileCount_OnlyDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only directories, no files
	createTestDir(t, tmpDir, "dir1")
	createTestDir(t, tmpDir, "dir2")
	subDir := createTestDir(t, tmpDir, "dir3")
	createTestDir(t, subDir, "subdir1")

	stats := NewFileStats()
	count, err := stats.GetFileCount(tmpDir)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should count 0 files (directories don't count)
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}
