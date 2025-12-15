package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MediaParser defines the interface for parsing and organising media files
type MediaParser interface {
	// Parse processes media files from source to target directory
	Parse(sourceDir, targetDir string, opts ParseOptions) error
	// ValidateDirectories checks if source and target directories exist
	ValidateDirectories(sourceDir, targetDir string) error
	// GetFileCount returns the number of files in a directory recursively
	GetFileCount(dir string) (int, error)
}

// ParseOptions holds configuration options for parsing
type ParseOptions struct {
	// CompressJPEGs enables JPEG compression
	CompressJPEGs bool
	// JPEGQuality is the quality level for JPEG compression (0-100)
	JPEGQuality int
	// TempDirName is the name of the temporary directory to use
	TempDirName string
}

// DefaultParseOptions returns the default parsing options
func DefaultParseOptions() ParseOptions {
	return ParseOptions{CompressJPEGs: true, JPEGQuality: 50, TempDirName: "tmp_image"}
}

// mediaParser implements the MediaParser interface
type mediaParser struct {
	compressor ImageCompressor
}

// NewMediaParser creates a new MediaParser instance
func NewMediaParser() MediaParser {
	return &mediaParser{
		compressor: NewImageCompressor(),
	}
}

// ValidateDirectories checks if source and target directories exist
func (p *mediaParser) ValidateDirectories(sourceDir, targetDir string) error {
	if info, err := os.Stat(sourceDir); err != nil || !info.IsDir() {
		return fmt.Errorf("SOURCE_DIR is not a valid directory: %s", sourceDir)
	}
	if info, err := os.Stat(targetDir); err != nil || !info.IsDir() {
		return fmt.Errorf("TARGET_DIR is not a valid directory: %s", targetDir)
	}
	return nil
}

// Parse processes media files from source to target directory
func (p *mediaParser) Parse(sourceDir, targetDir string, opts ParseOptions) error {
	if err := p.ValidateDirectories(sourceDir, targetDir); err != nil {
		return err
	}
	sourceDir = strings.TrimSuffix(sourceDir, "/")
	targetDir = strings.TrimSuffix(targetDir, "/")
	tmpTarget := filepath.Join(targetDir, opts.TempDirName)

	logger.Info("Creating temporary directory", "path", tmpTarget)
	if err := os.MkdirAll(tmpTarget, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpTarget)

	logger.Info("Copying media files", "source", sourceDir, "target", tmpTarget)
	if err := p.copyMediaFiles(sourceDir, tmpTarget); err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}

	if opts.CompressJPEGs {
		logger.Info("Compressing JPEGs", "quality", opts.JPEGQuality)
		if err := p.compressor.CompressDirectory(tmpTarget, opts.JPEGQuality); err != nil {
			return fmt.Errorf("failed to compress JPEGs: %w", err)
		}
	} else {
		logger.Info("Skipping JPEG compression")
	}

	logger.Info("Organising files by date")
	if err := p.organiseByDate(tmpTarget, targetDir); err != nil {
		return fmt.Errorf("failed to organise by date: %w", err)
	}

	logger.Info("Final organisation (videos and renaming)")
	if err := p.finalOrganisation(targetDir); err != nil {
		return fmt.Errorf("failed in final organisation: %w", err)
	}

	logger.Info("Processing complete")
	return nil
}

// copyMediaFiles copies MOV and JPG files from source subdirectories
func (p *mediaParser) copyMediaFiles(sourceDir, tmpTarget string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		prefix := entry.Name()
		subDir := filepath.Join(sourceDir, entry.Name())
		logger.Debug("Processing subdirectory", "dir", prefix)
		err := filepath.Walk(subDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			ext := strings.ToUpper(filepath.Ext(path))
			if ext == ".MOV" || ext == ".JPG" {
				destPath := filepath.Join(tmpTarget, fmt.Sprintf("%s-%s", prefix, filepath.Base(path)))
				logger.Debug("Copying file", "from", path, "to", destPath)
				if err := copyFilePreserveTime(path, destPath); err != nil {
					return fmt.Errorf("failed to copy %s: %w", path, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// copyFilePreserveTime copies a file and preserves its modification time
func copyFilePreserveTime(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return os.Chtimes(dst, time.Now(), srcInfo.ModTime())
}

// organiseByDate moves files to date-based directories
func (p *mediaParser) organiseByDate(tmpTarget, targetDir string) error {
	entries, err := os.ReadDir(tmpTarget)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(tmpTarget, entry.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		dirName := info.ModTime().Format("2006 01 January 02")
		destDir := filepath.Join(targetDir, dirName)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
		if err := os.Rename(filePath, filepath.Join(destDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// finalOrganisation organises videos into subdirectories and renames JPGs
func (p *mediaParser) finalOrganisation(targetDir string) error {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(targetDir, entry.Name())
		if err := p.organiseVideos(dirPath); err != nil {
			return err
		}
		if err := p.renameJPGs(dirPath, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

// organiseVideos moves MOV files to a videos subdirectory
func (p *mediaParser) organiseVideos(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.MOV"))
	if err != nil || len(files) == 0 {
		return err
	}
	videosDir := filepath.Join(dir, "videos")
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		return err
	}
	for _, file := range files {
		if err := os.Rename(file, filepath.Join(videosDir, filepath.Base(file))); err != nil {
			return err
		}
	}
	return nil
}

// renameJPGs renames JPG files with a sequential pattern
func (p *mediaParser) renameJPGs(dir, dirName string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.JPG"))
	if err != nil || len(files) == 0 {
		return err
	}
	sort.Strings(files)
	parts := strings.Fields(dirName)
	if len(parts) != 4 {
		return fmt.Errorf("unexpected directory name format: %s", dirName)
	}
	picsName := strings.Join(parts, "_")
	for i, file := range files {
		newPath := filepath.Join(dir, fmt.Sprintf("%s_%05d.jpg", picsName, i+1))
		if err := os.Rename(file, newPath); err != nil {
			return err
		}
	}
	return nil
}

// GetFileCount returns the number of files in a directory recursively
func (p *mediaParser) GetFileCount(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}
