package pics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createValidJPEG creates a minimal valid JPEG file for testing
func createValidJPEG(t *testing.T, dir, filename string) string {
	t.Helper()
	// Use the shared helper with current time as modification time
	return createValidJPEGWithDate(t, dir, filename, time.Now())
}

func TestExifWriter_WriteOriginalFileNameIfMissing_FirstTime(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createValidJPEG(t, tmpDir, "test_image.jpg")

	writer := NewExifWriter(createTestExiftool(t))
	written, err := writer.WriteOriginalFileNameIfMissing(testFile)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !written {
		t.Error("Expected field to be written on first call")
	}
}

func TestExifWriter_WriteOriginalFileNameIfMissing_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createValidJPEG(t, tmpDir, "test_image.jpg")

	writer := NewExifWriter(createTestExiftool(t))

	// First write
	written1, err := writer.WriteOriginalFileNameIfMissing(testFile)
	if err != nil {
		t.Fatalf("First write failed: %v", err)
	}
	if !written1 {
		t.Error("Expected field to be written on first call")
	}

	// Second write (should skip)
	written2, err := writer.WriteOriginalFileNameIfMissing(testFile)
	if err != nil {
		t.Errorf("Second write failed: %v", err)
	}
	if written2 {
		t.Error("Expected field to not be written on second call (already exists)")
	}
}

func TestExifWriter_WriteOriginalFileNameIfMissing_PreservesOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	originalName := "original_photo.jpg"
	testFile := createValidJPEG(t, tmpDir, originalName)

	writer := NewExifWriter(createTestExiftool(t))

	// Write the original filename
	_, err := writer.WriteOriginalFileNameIfMissing(testFile)
	if err != nil {
		t.Fatalf("Failed to write EXIF: %v", err)
	}

	// Rename the file
	newName := "renamed_photo.jpg"
	newPath := filepath.Join(tmpDir, newName)
	if err := os.Rename(testFile, newPath); err != nil {
		t.Fatalf("Failed to rename file: %v", err)
	}

	// Try to write again with new filename (should not overwrite)
	written, err := writer.WriteOriginalFileNameIfMissing(newPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if written {
		t.Error("Expected field to not be overwritten after rename")
	}

	// Verify the EXIF still contains the original filename
	et := createTestExiftool(t)
	fileInfos := et.ExtractMetadata(newPath)
	if len(fileInfos) == 0 {
		t.Fatal("No metadata found")
	}

	originalFileName, err := fileInfos[0].GetString(ExifOriginalFileName)
	if err != nil {
		t.Errorf("Failed to read %s: %v", ExifOriginalFileName, err)
	}

	if originalFileName != originalName {
		t.Errorf("Expected %s to be %s, got %s", ExifOriginalFileName, originalName, originalFileName)
	}
}

func TestExifWriter_WriteOriginalFileNameIfMissing_MultipleCalls(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := createValidJPEG(t, tmpDir, "photo.jpg")

	writer := NewExifWriter(createTestExiftool(t))

	// Multiple writes should all succeed but only first should write
	for i := 0; i < 3; i++ {
		written, err := writer.WriteOriginalFileNameIfMissing(testFile)
		if err != nil {
			t.Errorf("Call %d failed: %v", i+1, err)
		}
		if i == 0 && !written {
			t.Errorf("Call %d: expected field to be written", i+1)
		}
		if i > 0 && written {
			t.Errorf("Call %d: expected field to not be written (already exists)", i+1)
		}
	}
}

func TestExifWriter_WriteOriginalFileNameIfMissing_DifferentExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name     string
		filename string
	}{
		{"JPEG", "test.jpg"},
		{"JPEG uppercase", "test.JPG"},
		{"JPEG alternate", "test.jpeg"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := createValidJPEG(t, tmpDir, tc.filename)
			writer := NewExifWriter(createTestExiftool(t))

			written, err := writer.WriteOriginalFileNameIfMissing(testFile)
			if err != nil {
				t.Errorf("Failed for %s: %v", tc.filename, err)
			}
			if !written {
				t.Errorf("Expected field to be written for %s", tc.filename)
			}
		})
	}
}

func TestExifWriter_WriteOriginalFileNameIfMissing_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentFile := filepath.Join(tmpDir, "nonexistent.jpg")

	writer := NewExifWriter(createTestExiftool(t))
	_, err := writer.WriteOriginalFileNameIfMissing(nonexistentFile)

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestExifWriter_WriteOriginalFileNameIfMissing_SkipsVideoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	videoExtensions := []string{".mov", ".MOV", ".mp4", ".MP4", ".avi"}
	for _, ext := range videoExtensions {
		t.Run(ext, func(t *testing.T) {
			// Create a dummy file (doesn't need to be valid since we skip it)
			testFile := createFile(t, tmpDir, "video"+ext)
			writer := NewExifWriter(createTestExiftool(t))

			written, err := writer.WriteOriginalFileNameIfMissing(testFile)

			if err != nil {
				t.Errorf("Expected no error for video file, got: %v", err)
			}
			if written {
				t.Error("Expected video files to be skipped (not written)")
			}
		})
	}
}

func TestExifWriter_WriteOriginalFileNameIfMissing_InvalidJPEG(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with .jpg extension but invalid content
	testFile := createFile(t, tmpDir, "invalid.jpg")

	writer := NewExifWriter(createTestExiftool(t))
	_, err := writer.WriteOriginalFileNameIfMissing(testFile)

	// Should return an error because the file is not a valid JPEG
	if err == nil {
		t.Error("Expected error for invalid JPEG file")
	}
}
