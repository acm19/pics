package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/acm19/pics/apps/cli/completion"
	"github.com/acm19/pics/internal/logger"
	"github.com/acm19/pics/internal/pics"
	"github.com/barasher/go-exiftool"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "pics",
	Short:   "A Go application for organising and compressing photos and videos",
	Long:    `Pics helps you organise media files, compress images, and backup/restore to S3.`,
	Version: version,
}

var parseCmd = &cobra.Command{
	Use:   "parse SOURCE_DIR TARGET_DIR",
	Short: "Process and organise media files",
	Long:  `Copies media files from source subdirectories, optionally compresses JPEGs, and organises into date-based directories.`,
	Args:  cobra.ExactArgs(2),
	Run:   runParse,
}

var renameCmd = &cobra.Command{
	Use:   "rename DIRECTORY NAME",
	Short: "Rename a date-based directory and its images",
	Long:  `Renames a date-based directory (format: YYYY MM Month DD [current-name]) and updates all image filenames.`,
	Args:  cobra.ExactArgs(2),
	Run:   runRename,
}

var backupCmd = &cobra.Command{
	Use:   "backup SOURCE_DIR BUCKET",
	Short: "Backup directories to S3",
	Long:  `Creates tar.gz archives of each subdirectory and uploads to S3 with deduplication (MD5 hash comparison).`,
	Args:  cobra.ExactArgs(2),
	Run:   runBackup,
}

var restoreCmd = &cobra.Command{
	Use:   "restore BUCKET TARGET_DIR",
	Short: "Restore directories from S3",
	Long:  `Downloads and extracts backup archives from S3 with optional date-range filtering.`,
	Args:  cobra.ExactArgs(2),
	Run:   runRestore,
}

var (
	compressJPEGs bool
	jpegQuality   int
	maxConcurrent int
	fromFilter    string
	toFilter      string
)

func init() {
	// Parse command flags
	parseCmd.Flags().BoolVarP(&compressJPEGs, "compress", "c", true, "Enable JPEG compression")
	parseCmd.Flags().IntVarP(&jpegQuality, "rate", "r", 50, "JPEG compression quality (0-100)")

	// Backup command flags
	backupCmd.Flags().IntVarP(&maxConcurrent, "max-concurrent", "c", 5, "Maximum concurrent operations")

	// Restore command flags
	restoreCmd.Flags().IntVarP(&maxConcurrent, "max-concurrent", "c", 5, "Maximum concurrent operations")
	restoreCmd.Flags().StringVar(&fromFilter, "from", "", "Lower bound in format YYYY or MM/YYYY")
	restoreCmd.Flags().StringVar(&toFilter, "to", "", "Upper bound in format YYYY or MM/YYYY")

	// Add all subcommands
	rootCmd.AddCommand(parseCmd, renameCmd, backupCmd, restoreCmd)

	// Add autocomplete commands
	rootCmd.AddCommand(completion.NewInstallCmd(rootCmd))
	rootCmd.AddCommand(completion.NewUninstallCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runParse(cmd *cobra.Command, args []string) {
	sourceDir := args[0]
	targetDir := args[1]

	// Initialise exiftool for this command
	et, err := exiftool.NewExiftool()
	if err != nil {
		logger.Error("Failed to initialise exiftool", "error", err)
		os.Exit(1)
	}
	defer et.Close()

	fileStats := pics.NewFileStats()
	if err := fileStats.ValidateDirectories(sourceDir, targetDir); err != nil {
		logger.Error("Directory validation failed", "error", err)
		os.Exit(1)
	}

	opts := pics.DefaultParseOptions()
	opts.CompressJPEGs = compressJPEGs
	opts.JPEGQuality = jpegQuality

	sourceCount, err := fileStats.GetFileCount(sourceDir)
	if err != nil {
		logger.Error("Error counting source files", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting media parsing", "source", sourceDir, "target", targetDir)
	organiser := pics.NewFileOrganiser(et)
	exifWriter := pics.NewExifWriter(et)
	parser := pics.NewMediaParser("", organiser, exifWriter)
	if err := parser.Parse(sourceDir, targetDir, opts); err != nil {
		logger.Error("Parse failed", "error", err)
		os.Exit(1)
	}

	targetCount, err := fileStats.GetFileCount(targetDir)
	if err != nil {
		logger.Error("Error counting target files", "error", err)
		os.Exit(1)
	}

	if sourceCount != targetCount {
		logger.Error("File count mismatch", "source_files", sourceCount, "target_files", targetCount, "difference", targetCount-sourceCount)
		os.Exit(1)
	}

	logger.Info("Processing completed successfully", "files_processed", sourceCount, "verification", "source and target file counts match")
}

func runRename(cmd *cobra.Command, args []string) {
	directory := args[0]
	newName := args[1]

	// Initialise exiftool for this command
	et, err := exiftool.NewExiftool()
	if err != nil {
		logger.Error("Failed to initialise exiftool", "error", err)
		os.Exit(1)
	}
	defer et.Close()

	renamer := pics.NewDirectoryRenamer(et)
	if err := renamer.RenameDirectory(directory, newName); err != nil {
		logger.Error("Rename failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Rename completed successfully")
}

func runBackup(cmd *cobra.Command, args []string) {
	sourceDir := args[0]
	bucket := args[1]

	// Validate source directory exists
	if info, err := os.Stat(sourceDir); err != nil {
		logger.Error("Source directory does not exist", "directory", sourceDir, "error", err)
		os.Exit(1)
	} else if !info.IsDir() {
		logger.Error("Source path is not a directory", "path", sourceDir)
		os.Exit(1)
	}

	// Create backup instance
	ctx := context.Background()
	backup, err := pics.NewS3Backup(ctx)
	if err != nil {
		logger.Error("Failed to initialise backup", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting backup", "source", sourceDir, "bucket", bucket, "max_concurrent", maxConcurrent)
	if err := backup.BackupDirectories(ctx, sourceDir, bucket, maxConcurrent, nil); err != nil {
		logger.Error("Backup failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Backup completed successfully")
}

func runRestore(cmd *cobra.Command, args []string) {
	bucket := args[0]
	targetDir := args[1]

	// Parse filter
	var filter pics.RestoreFilter

	if fromFilter != "" {
		year, month, err := parseYearMonth(fromFilter)
		if err != nil {
			logger.Error("Invalid FROM value (expected YYYY or MM/YYYY)", "value", fromFilter, "error", err)
			os.Exit(1)
		}
		filter.FromYear = year
		filter.FromMonth = month
	}

	if toFilter != "" {
		year, month, err := parseYearMonth(toFilter)
		if err != nil {
			logger.Error("Invalid TO value (expected YYYY or MM/YYYY)", "value", toFilter, "error", err)
			os.Exit(1)
		}
		filter.ToYear = year
		filter.ToMonth = month
	}

	// Validate target directory exists
	if info, err := os.Stat(targetDir); err != nil {
		logger.Error("Target directory does not exist", "directory", targetDir, "error", err)
		os.Exit(1)
	} else if !info.IsDir() {
		logger.Error("Target path is not a directory", "path", targetDir)
		os.Exit(1)
	}

	// Create backup instance
	ctx := context.Background()
	backup, err := pics.NewS3Backup(ctx)
	if err != nil {
		logger.Error("Failed to initialise backup", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting restore", "bucket", bucket, "target", targetDir, "max_concurrent", maxConcurrent, "filter", filter)
	if err := backup.RestoreDirectories(ctx, bucket, targetDir, filter, maxConcurrent, nil); err != nil {
		logger.Error("Restore failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Restore completed successfully")
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
