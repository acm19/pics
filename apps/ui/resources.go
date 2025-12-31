package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed build/resources/windows/exiftool.exe
//go:embed build/resources/windows/jpegoptim.exe
//go:embed build/resources/darwin/exiftool
//go:embed build/resources/darwin/jpegoptim
//go:embed build/resources/darwin/lib
//go:embed build/resources/linux/exiftool
//go:embed build/resources/linux/jpegoptim
//go:embed build/resources/linux/lib
var resources embed.FS

// ExtractBinaries extracts the platform-specific exiftool and jpegoptim binaries
// to a temporary directory and returns their paths.
func ExtractBinaries() (exiftoolPath, jpegoptimPath string, err error) {
	// Create temp directory for extracted binaries
	tempDir := filepath.Join(os.TempDir(), "pics-ui-tools")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Determine platform
	platform := runtime.GOOS
	ext := ""
	if platform == "windows" {
		ext = ".exe"
	}

	// Extract exiftool
	exiftoolSrc := fmt.Sprintf("build/resources/%s/exiftool%s", platform, ext)
	exiftoolPath = filepath.Join(tempDir, "exiftool"+ext)
	if err := extractFile(exiftoolSrc, exiftoolPath); err != nil {
		return "", "", fmt.Errorf("failed to extract exiftool: %w", err)
	}

	// Make executable on Unix systems
	if platform != "windows" {
		if err := os.Chmod(exiftoolPath, 0755); err != nil {
			return "", "", fmt.Errorf("failed to make exiftool executable: %w", err)
		}

		// Extract lib directory for exiftool
		libSrc := fmt.Sprintf("build/resources/%s/lib", platform)
		libDest := filepath.Join(tempDir, "lib")
		if err := extractDir(libSrc, libDest); err != nil {
			return "", "", fmt.Errorf("failed to extract exiftool lib: %w", err)
		}
	}

	// Extract jpegoptim
	jpegoptimSrc := fmt.Sprintf("build/resources/%s/jpegoptim%s", platform, ext)
	jpegoptimPath = filepath.Join(tempDir, "jpegoptim"+ext)
	if err := extractFile(jpegoptimSrc, jpegoptimPath); err != nil {
		return "", "", fmt.Errorf("failed to extract jpegoptim: %w", err)
	}

	// Make executable on Unix systems
	if platform != "windows" {
		if err := os.Chmod(jpegoptimPath, 0755); err != nil {
			return "", "", fmt.Errorf("failed to make jpegoptim executable: %w", err)
		}
	}

	return exiftoolPath, jpegoptimPath, nil
}

// extractFile extracts a single file from the embedded filesystem to the destination path.
func extractFile(src, dst string) error {
	// Read from embedded FS
	data, err := resources.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read embedded file %s: %w", src, err)
	}

	// Write to destination
	file, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dst, err)
	}
	defer file.Close()

	if _, err := io.WriteString(file, string(data)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", dst, err)
	}

	return nil
}

// extractDir recursively extracts a directory from the embedded filesystem.
func extractDir(src, dst string) error {
	entries, err := resources.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read embedded dir %s: %w", src, err)
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create dir %s: %w", dst, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := extractDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := extractFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
