package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Shell represents a supported shell
type Shell string

const (
	Bash       Shell = "bash"
	Zsh        Shell = "zsh"
	Fish       Shell = "fish"
	Powershell Shell = "powershell"
)

// DetectShell detects the user's current shell from the SHELL environment variable
func DetectShell() (Shell, error) {
	// Check SHELL environment variable
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			return Powershell, nil
		}
		return "", fmt.Errorf("unable to detect shell: SHELL environment variable not set")
	}

	shellName := filepath.Base(shellPath)
	switch shellName {
	case "bash":
		return Bash, nil
	case "zsh":
		return Zsh, nil
	case "fish":
		return Fish, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}
}

// GetInstallPath returns the installation path for shell completion scripts
func GetInstallPath(shell Shell, home string) (string, error) {
	switch shell {
	case Bash:
		return filepath.Join(home, ".bash_completion.d", "pics"), nil
	case Zsh:
		return filepath.Join(home, ".zsh", "completion", "_pics"), nil
	case Fish:
		return filepath.Join(home, ".config", "fish", "completions", "pics.fish"), nil
	case Powershell:
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "Documents", "WindowsPowerShell", "Scripts", "pics.ps1"), nil
		}
		return "", fmt.Errorf("powershell not supported on %s", runtime.GOOS)
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}
