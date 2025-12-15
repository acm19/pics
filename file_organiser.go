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
	// OrganiseVideosAndRenameJPGs organises videos into subdirectories and renames JPGs sequentially
	OrganiseVideosAndRenameJPGs(targetDir string) error
}

// fileOrganiser implements the FileOrganiser interface
type fileOrganiser struct{}

// NewFileOrganiser creates a new FileOrganiser instance
func NewFileOrganiser() FileOrganiser {
	return &fileOrganiser{}
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

// OrganiseVideosAndRenameJPGs organises videos into subdirectories and renames JPGs sequentially
func (o *fileOrganiser) OrganiseVideosAndRenameJPGs(targetDir string) error {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(targetDir, entry.Name())
		if err := o.organiseVideos(dirPath); err != nil {
			return err
		}
		if err := o.renameJPGs(dirPath, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

// organiseVideos moves MOV files to a videos subdirectory
func (o *fileOrganiser) organiseVideos(dir string) error {
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
func (o *fileOrganiser) renameJPGs(dir, dirName string) error {
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
