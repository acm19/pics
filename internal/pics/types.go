package pics

// ParseOptions holds configuration options for parsing.
type ParseOptions struct {
	// CompressJPEGs enables JPEG compression.
	CompressJPEGs bool
	// JPEGQuality is the quality level for JPEG compression (0-100).
	JPEGQuality int
	// TempDirName is the name of the temporary directory to use.
	TempDirName string
	// MaxConcurrency is the maximum number of files to process concurrently (0 = unlimited).
	MaxConcurrency int
	// ProgressChan is an optional channel for receiving progress events.
	ProgressChan chan<- ProgressEvent
}

// DefaultParseOptions returns the default parsing options.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		CompressJPEGs:  true,
		JPEGQuality:    50,
		TempDirName:    "tmp_image",
		MaxConcurrency: 100,
		ProgressChan:   nil,
	}
}

// ProgressEvent represents a progress update during file processing operations.
type ProgressEvent struct {
	// Stage indicates the current processing stage ("copying", "compressing", "organising", "renaming").
	Stage string
	// Current is the number of items processed so far.
	Current int
	// Total is the total number of items to process.
	Total int
	// Message is a human-readable description of the current operation.
	Message string
	// File is the path of the file currently being processed.
	File string
}

// RestoreFilter defines the date range filter for restoring backups.
type RestoreFilter struct {
	// FromYear is the lower bound year (0 means no lower bound).
	FromYear int
	// FromMonth is the lower bound month (0 means January if FromYear is set).
	FromMonth int
	// ToYear is the upper bound year (0 means no upper bound).
	ToYear int
	// ToMonth is the upper bound month (0 means December if ToYear is set).
	ToMonth int
}
