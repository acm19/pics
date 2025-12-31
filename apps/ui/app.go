package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/acm19/pics/internal/logger"
	"github.com/acm19/pics/internal/pics"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx            context.Context
	exiftoolPath   string
	jpegoptimPath  string
	progressChan   chan pics.ProgressEvent
}

// NewApp creates a new App application struct
func NewApp(exiftoolPath, jpegoptimPath string) *App {
	return &App{
		exiftoolPath:  exiftoolPath,
		jpegoptimPath: jpegoptimPath,
		progressChan:  make(chan pics.ProgressEvent, 100),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.Info("Application started", "version", version)

	// Start progress event listener
	go a.listenForProgress()
}

// domReady is called after the front-end dom has been loaded
func (a *App) domReady(ctx context.Context) {
	logger.Debug("DOM ready")
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	logger.Info("Application shutting down")
	close(a.progressChan)
}

// listenForProgress listens for progress events and emits them to the frontend
func (a *App) listenForProgress() {
	for event := range a.progressChan {
		runtime.EventsEmit(a.ctx, "progress", map[string]any{
			"stage":   event.Stage,
			"current": event.Current,
			"total":   event.Total,
			"message": event.Message,
			"file":    event.File,
		})
	}
}

// ParseOptions holds options for the Parse operation
type ParseOptions struct {
	SourceDir      string `json:"sourceDir"`
	TargetDir      string `json:"targetDir"`
	CompressJPEGs  bool   `json:"compressJPEGs"`
	JPEGQuality    int    `json:"jpegQuality"`
	MaxConcurrency int    `json:"maxConcurrency"`
}

// Parse processes media files from source to target directory
func (a *App) Parse(opts ParseOptions) error {
	logger.Info("Starting parse operation", "source", opts.SourceDir, "target", opts.TargetDir)

	// Create file organiser with custom binary paths
	organiser := pics.NewFileOrganiserWithPaths(a.exiftoolPath)

	// Create media parser with custom binary paths
	parser := pics.NewMediaParserWithPaths(a.jpegoptimPath, organiser)

	// Create parse options with progress channel
	parseOpts := pics.ParseOptions{
		CompressJPEGs:  opts.CompressJPEGs,
		JPEGQuality:    opts.JPEGQuality,
		MaxConcurrency: opts.MaxConcurrency,
		TempDirName:    ".pics-temp",
		ProgressChan:   a.progressChan,
	}

	// Execute parse
	if err := parser.Parse(opts.SourceDir, opts.TargetDir, parseOpts); err != nil {
		logger.Error("Parse operation failed", "error", err)
		return err
	}

	logger.Info("Parse operation completed successfully")
	return nil
}

// BackupOptions holds options for the Backup operation
type BackupOptions struct {
	SourceDir string `json:"sourceDir"`
	Bucket    string `json:"bucket"`
}

// Backup creates tar.gz archives and uploads to S3
func (a *App) Backup(opts BackupOptions) error {
	logger.Info("Starting backup operation", "source", opts.SourceDir, "bucket", opts.Bucket)

	backup, err := pics.NewS3Backup(a.ctx)
	if err != nil {
		logger.Error("Failed to create S3 backup client", "error", err)
		return err
	}

	if err := backup.BackupDirectories(a.ctx, opts.SourceDir, opts.Bucket, 10, a.progressChan); err != nil {
		logger.Error("Backup operation failed", "error", err)
		return err
	}

	logger.Info("Backup operation completed successfully")
	return nil
}

// RestoreOptions holds options for the Restore operation
type RestoreOptions struct {
	Bucket     string `json:"bucket"`
	TargetDir  string `json:"targetDir"`
	FromFilter string `json:"fromFilter"`
	ToFilter   string `json:"toFilter"`
}

// Restore downloads and extracts archives from S3
func (a *App) Restore(opts RestoreOptions) error {
	logger.Info("Starting restore operation", "bucket", opts.Bucket, "target", opts.TargetDir, "from", opts.FromFilter, "to", opts.ToFilter)

	backup, err := pics.NewS3Backup(a.ctx)
	if err != nil {
		logger.Error("Failed to create S3 backup client", "error", err)
		return err
	}

	// Parse filter
	filter := pics.RestoreFilter{}
	if opts.FromFilter != "" {
		year, month, err := parseYearMonth(opts.FromFilter)
		if err != nil {
			return fmt.Errorf("invalid FROM filter (expected YYYY or MM/YYYY): %w", err)
		}
		filter.FromYear = year
		filter.FromMonth = month
	}
	if opts.ToFilter != "" {
		year, month, err := parseYearMonth(opts.ToFilter)
		if err != nil {
			return fmt.Errorf("invalid TO filter (expected YYYY or MM/YYYY): %w", err)
		}
		filter.ToYear = year
		filter.ToMonth = month
	}

	if err := backup.RestoreDirectories(a.ctx, opts.Bucket, opts.TargetDir, filter, 10, a.progressChan); err != nil {
		logger.Error("Restore operation failed", "error", err)
		return err
	}

	logger.Info("Restore operation completed successfully")
	return nil
}

// RenameOptions holds options for the Rename operation
type RenameOptions struct {
	Directory string `json:"directory"`
	NewName   string `json:"newName"`
}

// Rename renames a date-based directory and its images
func (a *App) Rename(opts RenameOptions) error {
	logger.Info("Starting rename operation", "directory", opts.Directory, "newName", opts.NewName)

	renamer := pics.NewDirectoryRenamer()
	if err := renamer.RenameDirectory(opts.Directory, opts.NewName); err != nil {
		logger.Error("Rename operation failed", "error", err)
		return err
	}

	logger.Info("Rename operation completed successfully")
	return nil
}

// SelectDirectory opens a directory selection dialog
func (a *App) SelectDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Directory",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// GetVersion returns the application version
func (a *App) GetVersion() string {
	return version
}

// parseYearMonth parses a date string in format "YYYY" or "MM/YYYY".
// Returns (year, month, error). Month is 0 if not specified.
func parseYearMonth(s string) (int, int, error) {
	parts := strings.Split(s, "/")

	if len(parts) == 1 {
		// Format: YYYY
		year, err := strconv.Atoi(parts[0])
		if err != nil || year < 1000 || year > 9999 {
			return 0, 0, fmt.Errorf("invalid year: %s", parts[0])
		}
		return year, 0, nil
	} else if len(parts) == 2 {
		// Format: MM/YYYY
		month, err := strconv.Atoi(parts[0])
		if err != nil || month < 1 || month > 12 {
			return 0, 0, fmt.Errorf("invalid month (must be 1-12): %s", parts[0])
		}
		year, err := strconv.Atoi(parts[1])
		if err != nil || year < 1000 || year > 9999 {
			return 0, 0, fmt.Errorf("invalid year: %s", parts[1])
		}
		return year, month, nil
	}

	return 0, 0, fmt.Errorf("invalid format (expected YYYY or MM/YYYY): %s", s)
}
