package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewUninstallCmd creates the uninstall-autocomplete command
func NewUninstallCmd() *cobra.Command {
	var shellFlag string

	cmd := &cobra.Command{
		Use:   "uninstall-autocomplete",
		Short: "Uninstall shell completion for pics",
		Long:  `Uninstall shell completion script for the pics CLI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			return runUninstall(shellFlag, home)
		},
	}

	cmd.Flags().StringVarP(&shellFlag, "shell", "s", "", "Shell to uninstall completion from (bash, zsh, fish, powershell). Auto-detected if not specified.")

	return cmd
}

func runUninstall(shellFlag string, home string) error {
	// Determine shell
	var shell Shell
	var err error

	if shellFlag != "" {
		shell = Shell(shellFlag)
	} else {
		shell, err = DetectShell()
		if err != nil {
			return fmt.Errorf("failed to detect shell: %w\nSpecify shell explicitly with --shell flag", err)
		}
	}

	// Get installation path
	installPath, err := GetInstallPath(shell, home)
	if err != nil {
		return err
	}

	// Check if completion is installed
	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return fmt.Errorf("completion not installed for %s (expected at %s)", shell, installPath)
	}

	// Disable auto-load for bash
	if shell == Bash {
		bashCompletionFile := filepath.Join(home, ".bash_completion")
		if err := disableBashAutoLoad(bashCompletionFile, installPath); err != nil {
			// Non-fatal: warn but continue
			fmt.Printf("Warning: could not disable auto-load: %v\n", err)
		}
	}

	// Remove completion file
	if err := os.Remove(installPath); err != nil {
		return fmt.Errorf("failed to remove completion file: %w", err)
	}

	fmt.Printf("Shell completion uninstalled successfully for %s\n", shell)
	fmt.Printf("Removed: %s\n", installPath)

	// Print shell-specific cleanup instructions
	printCleanupInstructions(shell)

	return nil
}

func printCleanupInstructions(shell Shell) {
	switch shell {
	case Bash:
		fmt.Println("\nRestart your shell to complete removal.")

	case Zsh:
		fmt.Println("\nRestart your shell to complete removal.")

	case Fish:
		fmt.Println("\nRestart fish to complete removal: exec fish")

	case Powershell:
		fmt.Println("\nYou may want to remove the source line from your PowerShell profile")
	}
}

// disableBashAutoLoad removes the source line from the bash completion file.
// bashCompletionFile is injected for testability (production: ~/.bash_completion).
func disableBashAutoLoad(bashCompletionFile, installPath string) error {
	content, err := os.ReadFile(bashCompletionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to remove
		}
		return err
	}

	// Remove any line containing the install path (handles different source syntaxes)
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.Contains(line, installPath) {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(bashCompletionFile, []byte(strings.Join(newLines, "\n")), 0644)
}
