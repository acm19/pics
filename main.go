package main

import (
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 3 {
		logger.Error("Invalid arguments", "usage", os.Args[0]+" SOURCE_DIR TARGET_DIR")
		os.Exit(1)
	}

	sourceDir := os.Args[1]
	targetDir := os.Args[2]

	parser := NewMediaParser()
	if err := parser.ValidateDirectories(sourceDir, targetDir); err != nil {
		logger.Error("Directory validation failed", "error", err)
		os.Exit(1)
	}

	opts := DefaultParseOptions()
	if rate := os.Getenv("RATE"); rate != "" {
		quality, err := strconv.Atoi(rate)
		if err != nil {
			logger.Error("Invalid RATE value", "rate", rate, "error", err)
			os.Exit(1)
		}
		opts.JPEGQuality = quality
	}

	sourceCount, err := parser.GetFileCount(sourceDir)
	if err != nil {
		logger.Error("Error counting source files", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting media parsing", "source", sourceDir, "target", targetDir)
	if err := parser.Parse(sourceDir, targetDir, opts); err != nil {
		logger.Error("Parse failed", "error", err)
		os.Exit(1)
	}

	targetCount, err := parser.GetFileCount(targetDir)
	if err != nil {
		logger.Error("Error counting target files", "error", err)
		os.Exit(1)
	}

	logger.Info("Summary", "source_files", sourceCount, "target_files", targetCount)
}
