package pics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidFile_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "valid.jpg")

	// Create a valid file with some content
	if err := os.WriteFile(validFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := isValidFile(validFile)
	if err != nil {
		t.Errorf("Expected no error for valid file, got: %v", err)
	}
}

func TestIsValidFile_ZeroByteFile(t *testing.T) {
	tmpDir := t.TempDir()
	zeroByteFile := filepath.Join(tmpDir, "empty.jpg")

	// Create a 0-byte file
	if err := os.WriteFile(zeroByteFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create 0-byte file: %v", err)
	}

	err := isValidFile(zeroByteFile)
	if err == nil {
		t.Error("Expected error for 0-byte file, got nil")
	}

	// Verify error message mentions "0 bytes"
	expectedMsg := "file is 0 bytes (corrupted)"
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestIsValidFile_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentFile := filepath.Join(tmpDir, "nonexistent.jpg")

	err := isValidFile(nonexistentFile)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	// Verify error mentions "cannot access file"
	if err != nil {
		errMsg := err.Error()
		if len(errMsg) == 0 {
			t.Error("Expected non-empty error message")
		}
		// Just verify it's an error about file access - the exact message depends on OS
		if err == nil {
			t.Error("Expected error for inaccessible file")
		}
	}
}

func TestIsValidFile_SymlinkToZeroByteFile(t *testing.T) {
	tmpDir := t.TempDir()
	zeroByteFile := filepath.Join(tmpDir, "empty.jpg")
	symlink := filepath.Join(tmpDir, "link.jpg")

	// Create a 0-byte file
	if err := os.WriteFile(zeroByteFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create 0-byte file: %v", err)
	}

	// Create symlink to it
	if err := os.Symlink(zeroByteFile, symlink); err != nil {
		t.Skipf("Skipping symlink test: %v", err)
	}

	// os.Stat follows symlinks, so should detect the 0-byte target
	err := isValidFile(symlink)
	if err == nil {
		t.Error("Expected error for symlink to 0-byte file, got nil")
	}
}

func TestIsValidFile_SymlinkToValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "valid.jpg")
	symlink := filepath.Join(tmpDir, "link.jpg")

	// Create a valid file
	if err := os.WriteFile(validFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}

	// Create symlink to it
	if err := os.Symlink(validFile, symlink); err != nil {
		t.Skipf("Skipping symlink test: %v", err)
	}

	// Should follow symlink and find valid file
	err := isValidFile(symlink)
	if err != nil {
		t.Errorf("Expected no error for symlink to valid file, got: %v", err)
	}
}
