package pics

import (
	"testing"

	"github.com/barasher/go-exiftool"
)

// createTestExiftool creates an exiftool instance for testing and ensures cleanup
func createTestExiftool(t *testing.T) *exiftool.Exiftool {
	t.Helper()
	et, err := exiftool.NewExiftool()
	if err != nil {
		t.Fatalf("Failed to create exiftool: %v", err)
	}
	t.Cleanup(func() { et.Close() })
	return et
}
