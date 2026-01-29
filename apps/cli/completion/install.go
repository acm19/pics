package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewInstallCmd creates the install-autocomplete command
func NewInstallCmd(rootCmd *cobra.Command) *cobra.Command {
	var shellFlag string

	cmd := &cobra.Command{
		Use:   "install-autocomplete",
		Short: "Install shell completion for pics",
		Long: `Install shell completion for the pics CLI.

Automatically detects your shell and installs the appropriate completion script.
Supports bash, zsh, fish, and powershell.

The completion script enables tab completion for:
- Commands (parse, rename, backup, restore)
- Flags (--compress, --rate, --max-concurrent, --from, --to)
- File paths and directories`,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			return runInstall(rootCmd, shellFlag, home)
		},
	}

	cmd.Flags().StringVarP(&shellFlag, "shell", "s", "", "Shell to install completion for (bash, zsh, fish, powershell). Auto-detected if not specified.")

	return cmd
}

func runInstall(rootCmd *cobra.Command, shellFlag string, home string) error {
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

	// Create directory if needed
	dir := filepath.Dir(installPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create completion directory %s: %w", dir, err)
	}

	// Generate and write completion script
	if err := writeCompletionScript(rootCmd, shell, installPath); err != nil {
		return err
	}

	// Enable auto-load for bash
	if shell == Bash {
		bashCompletionFile := filepath.Join(home, ".bash_completion")
		if err := enableBashAutoLoad(bashCompletionFile, installPath); err != nil {
			// Non-fatal: warn but continue
			fmt.Printf("Warning: could not enable auto-load: %v\n", err)
		}
	}

	fmt.Printf("Shell completion installed successfully for %s\n", shell)
	fmt.Printf("Completion script location: %s\n", installPath)

	// Print shell-specific activation instructions
	printActivationInstructions(shell, installPath)

	return nil
}

func writeCompletionScript(rootCmd *cobra.Command, shell Shell, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create completion file: %w", err)
	}
	defer file.Close()

	switch shell {
	case Bash:
		return rootCmd.GenBashCompletionV2(file, true)
	case Zsh:
		return rootCmd.GenZshCompletion(file)
	case Fish:
		return rootCmd.GenFishCompletion(file, true)
	case Powershell:
		return rootCmd.GenPowerShellCompletionWithDesc(file)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func printActivationInstructions(shell Shell, installPath string) {
	switch shell {
	case Bash:
		fmt.Println("\nCompletion is now active. Open a new terminal to use it.")

	case Zsh:
		fmt.Println("\nTo activate completion, ensure this is in your ~/.zshrc:")
		fmt.Printf("  fpath=(%s $fpath)\n", filepath.Dir(installPath))
		fmt.Println("  autoload -Uz compinit && compinit")
		fmt.Println("\nThen restart your shell.")

	case Fish:
		fmt.Println("\nCompletion is automatically available in new fish sessions.")
		fmt.Println("Run 'exec fish' to activate in the current session.")

	case Powershell:
		fmt.Println("\nTo activate completion, add this to your PowerShell profile:")
		fmt.Printf("  . %s\n", installPath)
	}
}

// enableBashAutoLoad adds a source line to the bash completion file.
// bashCompletionFile is injected for testability (production: ~/.bash_completion).
func enableBashAutoLoad(bashCompletionFile, installPath string) error {
	sourceLine := fmt.Sprintf("source %s", installPath)

	// Read existing content (if any)
	content, _ := os.ReadFile(bashCompletionFile)

	// Check if already present anywhere in file (idempotent)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, installPath) {
			return nil // Already configured
		}
	}

	// Append source line
	f, err := os.OpenFile(bashCompletionFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline if file has content and doesn't end with newline
	if len(content) > 0 && content[len(content)-1] != '\n' {
		f.WriteString("\n")
	}

	_, err = f.WriteString(sourceLine + "\n")
	return err
}
