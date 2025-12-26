package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileOrganiser defines the interface for organising files
type FileOrganiser interface {
	// OrganiseByDate moves files to date-based directories
	OrganiseByDate(sourceDir, targetDir string) error
	// OrganiseVideosAndRenameImages organises videos into subdirectories and renames images sequentially
	OrganiseVideosAndRenameImages(targetDir string) error
}

// fileOrganiser implements the FileOrganiser interface
type fileOrganiser struct {
	dateExtractor *AggregatedFileDateExtractor
	extensions    Extensions
}

// NewFileOrganiser creates a new FileOrganiser instance
func NewFileOrganiser() FileOrganiser {
	return &fileOrganiser{
		dateExtractor: NewFileDateExtractor(),
		extensions:    NewExtensions(),
	}
}

// OrganiseByDate moves files to date-based directories
func (o *fileOrganiser) OrganiseByDate(sourceDir, targetDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(sourceDir, entry.Name())

		// Get file date from EXIF if available, otherwise use ModTime
		fileDate, err := o.dateExtractor.GetFileDate(filePath)
		if err != nil {
			return err
		}

		dirName := fileDate.Format("2006 01 January 02")
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

// OrganiseVideosAndRenameImages organises videos into subdirectories and renames images sequentially
func (o *fileOrganiser) OrganiseVideosAndRenameImages(targetDir string) error {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(targetDir, entry.Name())
		logger.Debug("Organising file %s/%s", dirPath, entry.Name())
		if err := o.organiseVideos(dirPath, entry.Name()); err != nil {
			return err
		}
		if err := o.renameImages(dirPath, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

// organiseVideos moves video files to a videos subdirectory and renames them sequentially
func (o *fileOrganiser) organiseVideos(dir string, dirName string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	videoFiles := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		if o.extensions.IsVideo(filePath) {
			videoFiles = append(videoFiles, filePath)
		}
	}

	if len(videoFiles) == 0 {
		return nil
	}
	videosDir := filepath.Join(dir, "videos")
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		return err
	}
	sort.Strings(videoFiles)
	parts := strings.Fields(dirName)
	if len(parts) != 4 {
		return fmt.Errorf("unexpected directory name format: %s", dirName)
	}
	videosName := strings.Join(parts, "_")
	for i, file := range videoFiles {
		ext := filepath.Ext(file)
		newPath := filepath.Join(videosDir, fmt.Sprintf("%s_%05d%s", videosName, i+1, ext))
		if err := os.Rename(file, newPath); err != nil {
			return err
		}
	}
	return nil
}

// renameImages renames image files with a sequential pattern
func (o *fileOrganiser) renameImages(dir, dirName string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	imageFiles := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		if o.extensions.IsImage(filePath) {
			imageFiles = append(imageFiles, filePath)
		}
	}

	if len(imageFiles) == 0 {
		return nil
	}
	sort.Strings(imageFiles)
	parts := strings.Fields(dirName)
	if len(parts) != 4 {
		return fmt.Errorf("unexpected directory name format: %s", dirName)
	}
	picsName := strings.Join(parts, "_")
	for i, file := range imageFiles {
		ext := strings.ToLower(filepath.Ext(file))
		newPath := filepath.Join(dir, fmt.Sprintf("%s_%05d%s", picsName, i+1, ext))
		if err := os.Rename(file, newPath); err != nil {
			return err
		}
	}
	return nil
}
