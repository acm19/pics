package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	fileRenamer   FileRenamer
}

// NewFileOrganiser creates a new FileOrganiser instance
func NewFileOrganiser() FileOrganiser {
	return &fileOrganiser{
		dateExtractor: NewFileDateExtractor(),
		extensions:    NewExtensions(),
		fileRenamer:   NewFileRenamer(),
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
	parts := strings.Fields(dirName)
	if len(parts) != 4 {
		return fmt.Errorf("unexpected directory name format: %s", dirName)
	}
	videosName := strings.Join(parts, "_")
	videosDir := filepath.Join(dir, "videos")
	_, err := o.fileRenamer.MoveAndRenameFilesWithPattern(dir, videosDir, videosName, o.extensions.IsVideo)
	return err
}

// renameImages renames image files with a sequential pattern
func (o *fileOrganiser) renameImages(dir, dirName string) error {
	parts := strings.Fields(dirName)
	if len(parts) != 4 {
		return fmt.Errorf("unexpected directory name format: %s", dirName)
	}
	picsName := strings.Join(parts, "_")
	_, err := o.fileRenamer.RenameFilesWithPattern(dir, picsName, o.extensions.IsImage)
	return err
}
