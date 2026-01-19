package pics

import (
	"fmt"
	"os"
)

// isValidFile checks if a file exists and is not empty (0 bytes).
// Returns an error if the file cannot be accessed or is 0 bytes.
func isValidFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("file is 0 bytes (corrupted)")
	}
	return nil
}
