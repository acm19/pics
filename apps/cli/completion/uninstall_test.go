package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewUninstallCmd(t *testing.T) {
	cmd := NewUninstallCmd()

	if cmd.Use != "uninstall-autocomplete" {
		t.Errorf("NewUninstallCmd().Use = %v, want uninstall-autocomplete", cmd.Use)
	}

	// Verify shell flag exists
	shellFlag := cmd.Flags().Lookup("shell")
	if shellFlag == nil {
		t.Error("NewUninstallCmd() should have --shell flag")
	}

	shorthand := cmd.Flags().ShorthandLookup("s")
	if shorthand == nil {
		t.Error("NewUninstallCmd() should have -s shorthand for --shell flag")
	}
}

func TestRunUninstall_InvalidShell(t *testing.T) {
	tmpDir := t.TempDir()

	err := runUninstall("invalidshell", tmpDir)
	if err == nil {
		t.Error("runUninstall() should return error for invalid shell")
	}

	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("runUninstall() error = %v, want error containing 'unsupported shell'", err)
	}
}

func TestRunUninstall_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	err := runUninstall("fish", tmpDir)
	if err == nil {
		t.Error("runUninstall() should return error when completion not installed")
	}

	if !strings.Contains(err.Error(), "completion not installed") {
		t.Errorf("runUninstall() error = %v, want error containing 'completion not installed'", err)
	}
}

func TestRunUninstall_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// First install completion
	rootCmd := &cobra.Command{
		Use:   "pics",
		Short: "Test command",
	}

	err := runInstall(rootCmd, "fish", tmpDir)
	if err != nil {
		t.Fatalf("runInstall() error = %v, want nil", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, ".config", "fish", "completions", "pics.fish")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("runInstall() did not create completion file at %s", expectedPath)
	}

	// Now uninstall
	err = runUninstall("fish", tmpDir)
	if err != nil {
		t.Fatalf("runUninstall() error = %v, want nil", err)
	}

	// Verify file was removed
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("runUninstall() did not remove completion file at %s", expectedPath)
	}
}

func TestRunUninstall_Zsh(t *testing.T) {
	tmpDir := t.TempDir()

	// First install completion
	rootCmd := &cobra.Command{
		Use:   "pics",
		Short: "Test command",
	}

	err := runInstall(rootCmd, "zsh", tmpDir)
	if err != nil {
		t.Fatalf("runInstall(zsh) error = %v, want nil", err)
	}

	expectedPath := filepath.Join(tmpDir, ".zsh", "completion", "_pics")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("runInstall(zsh) did not create file at %s", expectedPath)
	}

	// Now uninstall
	err = runUninstall("zsh", tmpDir)
	if err != nil {
		t.Fatalf("runUninstall(zsh) error = %v, want nil", err)
	}

	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("runUninstall(zsh) did not remove file at %s", expectedPath)
	}
}

func TestRunUninstall_Bash(t *testing.T) {
	tmpDir := t.TempDir()

	// First install completion
	rootCmd := &cobra.Command{
		Use:   "pics",
		Short: "Test command",
	}

	err := runInstall(rootCmd, "bash", tmpDir)
	if err != nil {
		t.Fatalf("runInstall(bash) error = %v, want nil", err)
	}

	expectedPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("runInstall(bash) did not create file at %s", expectedPath)
	}

	// Now uninstall
	err = runUninstall("bash", tmpDir)
	if err != nil {
		t.Fatalf("runUninstall(bash) error = %v, want nil", err)
	}

	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("runUninstall(bash) did not remove file at %s", expectedPath)
	}
}

func TestDisableBashAutoLoad(t *testing.T) {
	tmpDir := t.TempDir()
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")
	installPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")

	// Create file with the source line
	content := "# existing config\nsource " + installPath + "\nsource /other/completion\n"
	if err := os.WriteFile(bashCompletionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write bash completion file: %v", err)
	}

	err := disableBashAutoLoad(bashCompletionFile, installPath)
	if err != nil {
		t.Fatalf("disableBashAutoLoad() error = %v", err)
	}

	newContent, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}

	// Verify the source line was removed
	if strings.Contains(string(newContent), installPath) {
		t.Errorf("source line was not removed, got:\n%s", string(newContent))
	}

	// Verify other content is preserved
	if !strings.Contains(string(newContent), "# existing config") {
		t.Error("existing comment was not preserved")
	}
	if !strings.Contains(string(newContent), "source /other/completion") {
		t.Error("other source line was not preserved")
	}
}

func TestDisableBashAutoLoad_FileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")
	installPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")

	// File doesn't exist - should not error
	err := disableBashAutoLoad(bashCompletionFile, installPath)
	if err != nil {
		t.Fatalf("disableBashAutoLoad() error = %v, want nil for non-existent file", err)
	}
}

func TestDisableBashAutoLoad_NoMatchingLine(t *testing.T) {
	tmpDir := t.TempDir()
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")
	installPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")

	// Create file without the source line
	content := "# existing config\nsource /other/completion\n"
	if err := os.WriteFile(bashCompletionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write bash completion file: %v", err)
	}

	err := disableBashAutoLoad(bashCompletionFile, installPath)
	if err != nil {
		t.Fatalf("disableBashAutoLoad() error = %v", err)
	}

	newContent, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}

	// Verify content is unchanged
	if !strings.Contains(string(newContent), "# existing config") {
		t.Error("existing content was modified")
	}
	if !strings.Contains(string(newContent), "source /other/completion") {
		t.Error("existing source line was modified")
	}
}

func TestRunUninstall_BashDisablesAutoLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// First install completion
	rootCmd := &cobra.Command{
		Use:   "pics",
		Short: "Test command",
	}

	err := runInstall(rootCmd, "bash", tmpDir)
	if err != nil {
		t.Fatalf("runInstall(bash) error = %v, want nil", err)
	}

	expectedPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")

	// Verify auto-load was enabled
	content, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}
	if !strings.Contains(string(content), expectedPath) {
		t.Fatalf("runInstall(bash) did not enable auto-load")
	}

	// Now uninstall
	err = runUninstall("bash", tmpDir)
	if err != nil {
		t.Fatalf("runUninstall(bash) error = %v, want nil", err)
	}

	// Verify completion file was removed
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("runUninstall(bash) did not remove file at %s", expectedPath)
	}

	// Verify auto-load was disabled
	content, err = os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}
	if strings.Contains(string(content), expectedPath) {
		t.Errorf("runUninstall(bash) did not disable auto-load, file contains:\n%s", string(content))
	}
}
