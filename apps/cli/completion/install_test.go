package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewInstallCmd(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	cmd := NewInstallCmd(rootCmd)

	if cmd.Use != "install-autocomplete" {
		t.Errorf("NewInstallCmd().Use = %v, want install-autocomplete", cmd.Use)
	}

	// Verify shell flag exists
	shellFlag := cmd.Flags().Lookup("shell")
	if shellFlag == nil {
		t.Error("NewInstallCmd() should have --shell flag")
	}

	shorthand := cmd.Flags().ShorthandLookup("s")
	if shorthand == nil {
		t.Error("NewInstallCmd() should have -s shorthand for --shell flag")
	}
}

func TestRunInstall_InvalidShell(t *testing.T) {
	tmpDir := t.TempDir()

	rootCmd := &cobra.Command{
		Use: "test",
	}

	err := runInstall(rootCmd, "invalidshell", tmpDir)
	if err == nil {
		t.Error("runInstall() should return error for invalid shell")
	}

	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("runInstall() error = %v, want error containing 'unsupported shell'", err)
	}
}

func TestRunInstall_ValidShell(t *testing.T) {
	tmpDir := t.TempDir()

	rootCmd := &cobra.Command{
		Use:   "pics",
		Short: "Test command",
	}
	rootCmd.AddCommand(&cobra.Command{
		Use:   "parse",
		Short: "Parse command",
	})

	// Test installation for fish (uses simple user directory structure)
	err := runInstall(rootCmd, "fish", tmpDir)
	if err != nil {
		t.Fatalf("runInstall() error = %v, want nil", err)
	}

	// Verify file was created
	expectedPath := filepath.Join(tmpDir, ".config", "fish", "completions", "pics.fish")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("runInstall() did not create completion file at %s", expectedPath)
	}

	// Verify file has content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read completion file: %v", err)
	}
	if len(content) == 0 {
		t.Error("runInstall() created empty completion file")
	}
}

func TestRunInstall_Zsh(t *testing.T) {
	tmpDir := t.TempDir()

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
		t.Errorf("runInstall(zsh) did not create completion file at %s", expectedPath)
	}
}

func TestRunInstall_Bash(t *testing.T) {
	tmpDir := t.TempDir()

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
		t.Errorf("runInstall(bash) did not create completion file at %s", expectedPath)
	}
}

func TestWriteCompletionScript(t *testing.T) {
	tmpDir := t.TempDir()

	rootCmd := &cobra.Command{
		Use:   "pics",
		Short: "Test command",
	}
	rootCmd.AddCommand(&cobra.Command{
		Use:   "parse",
		Short: "Parse command",
	})

	tests := []struct {
		name     string
		shell    Shell
		filename string
	}{
		{"bash completion", Bash, "pics.bash"},
		{"zsh completion", Zsh, "_pics"},
		{"fish completion", Fish, "pics.fish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.filename)
			err := writeCompletionScript(rootCmd, tt.shell, path)
			if err != nil {
				t.Fatalf("writeCompletionScript() error = %v", err)
			}

			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read completion file: %v", err)
			}

			if len(content) == 0 {
				t.Error("writeCompletionScript() created empty file")
			}

			// Verify the script contains the command name
			if !strings.Contains(string(content), "pics") {
				t.Error("writeCompletionScript() output does not contain 'pics'")
			}
		})
	}
}

func TestWriteCompletionScript_InvalidShell(t *testing.T) {
	tmpDir := t.TempDir()

	rootCmd := &cobra.Command{
		Use: "test",
	}

	path := filepath.Join(tmpDir, "test-completion")
	err := writeCompletionScript(rootCmd, Shell("invalid"), path)
	if err == nil {
		t.Error("writeCompletionScript() should return error for invalid shell")
	}
}

func TestEnableBashAutoLoad(t *testing.T) {
	tmpDir := t.TempDir()
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")
	installPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")

	// Test: adds source line to new file
	err := enableBashAutoLoad(bashCompletionFile, installPath)
	if err != nil {
		t.Fatalf("enableBashAutoLoad() error = %v", err)
	}

	content, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}

	expectedLine := "source " + installPath
	if !strings.Contains(string(content), expectedLine) {
		t.Errorf("expected source line %q in file, got %q", expectedLine, string(content))
	}

	// Test: idempotent - running again doesn't duplicate
	err = enableBashAutoLoad(bashCompletionFile, installPath)
	if err != nil {
		t.Fatalf("enableBashAutoLoad() second call error = %v", err)
	}

	content, err = os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}

	// Count occurrences of the source line
	count := strings.Count(string(content), expectedLine)
	if count != 1 {
		t.Errorf("expected exactly 1 source line, got %d in:\n%s", count, string(content))
	}
}

func TestEnableBashAutoLoad_ExistingContent(t *testing.T) {
	tmpDir := t.TempDir()
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")
	installPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")

	// Pre-populate with existing content (without trailing newline)
	existingContent := "# existing config\nsource /other/completion"
	if err := os.WriteFile(bashCompletionFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing content: %v", err)
	}

	err := enableBashAutoLoad(bashCompletionFile, installPath)
	if err != nil {
		t.Fatalf("enableBashAutoLoad() error = %v", err)
	}

	content, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}

	// Verify existing content is preserved
	if !strings.Contains(string(content), "# existing config") {
		t.Error("existing content was not preserved")
	}
	if !strings.Contains(string(content), "source /other/completion") {
		t.Error("existing source line was not preserved")
	}

	// Verify new source line was added
	expectedLine := "source " + installPath
	if !strings.Contains(string(content), expectedLine) {
		t.Errorf("expected source line %q in file, got %q", expectedLine, string(content))
	}
}

func TestEnableBashAutoLoad_ExistingContentWithTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")
	installPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")

	// Pre-populate with existing content (with trailing newline)
	existingContent := "# existing config\nsource /other/completion\n"
	if err := os.WriteFile(bashCompletionFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing content: %v", err)
	}

	err := enableBashAutoLoad(bashCompletionFile, installPath)
	if err != nil {
		t.Fatalf("enableBashAutoLoad() error = %v", err)
	}

	content, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}

	// Verify no double newlines were created
	if strings.Contains(string(content), "\n\nsource "+installPath) {
		t.Error("double newline was created before source line")
	}
}

func TestRunInstall_BashEnablesAutoLoad(t *testing.T) {
	tmpDir := t.TempDir()

	rootCmd := &cobra.Command{
		Use:   "pics",
		Short: "Test command",
	}

	err := runInstall(rootCmd, "bash", tmpDir)
	if err != nil {
		t.Fatalf("runInstall(bash) error = %v, want nil", err)
	}

	// Verify completion script was created
	expectedPath := filepath.Join(tmpDir, ".bash_completion.d", "pics")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("runInstall(bash) did not create completion file at %s", expectedPath)
	}

	// Verify auto-load was enabled
	bashCompletionFile := filepath.Join(tmpDir, ".bash_completion")
	content, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		t.Fatalf("Failed to read bash completion file: %v", err)
	}

	expectedLine := "source " + expectedPath
	if !strings.Contains(string(content), expectedLine) {
		t.Errorf("expected source line %q in .bash_completion, got %q", expectedLine, string(content))
	}
}
