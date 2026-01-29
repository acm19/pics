package completion

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestDetectShell(t *testing.T) {
	originalShell := os.Getenv("SHELL")
	defer func() {
		if originalShell != "" {
			os.Setenv("SHELL", originalShell)
		} else {
			os.Unsetenv("SHELL")
		}
	}()

	tests := []struct {
		name      string
		shellEnv  string
		want      Shell
		wantErr   bool
		skipOnWin bool
	}{
		{
			name:      "detect bash",
			shellEnv:  "/bin/bash",
			want:      Bash,
			wantErr:   false,
			skipOnWin: true,
		},
		{
			name:      "detect zsh",
			shellEnv:  "/usr/bin/zsh",
			want:      Zsh,
			wantErr:   false,
			skipOnWin: true,
		},
		{
			name:      "detect fish",
			shellEnv:  "/usr/local/bin/fish",
			want:      Fish,
			wantErr:   false,
			skipOnWin: true,
		},
		{
			name:      "unsupported shell",
			shellEnv:  "/bin/tcsh",
			want:      "",
			wantErr:   true,
			skipOnWin: true,
		},
		{
			name:      "empty shell env on non-windows",
			shellEnv:  "",
			want:      "",
			wantErr:   true,
			skipOnWin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("Skipping test on Windows")
			}

			if tt.shellEnv != "" {
				os.Setenv("SHELL", tt.shellEnv)
			} else {
				os.Unsetenv("SHELL")
			}

			got, err := DetectShell()
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectShell() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DetectShell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInstallPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		shell       Shell
		wantContain string
		wantErr     bool
	}{
		{
			name:        "bash completion path contains pics",
			shell:       Bash,
			wantContain: "pics",
			wantErr:     false,
		},
		{
			name:        "zsh completion path",
			shell:       Zsh,
			wantContain: "_pics",
			wantErr:     false,
		},
		{
			name:        "fish completion path",
			shell:       Fish,
			wantContain: "pics.fish",
			wantErr:     false,
		},
		{
			name:        "unsupported shell",
			shell:       Shell("tcsh"),
			wantContain: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInstallPath(tt.shell, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstallPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !strings.Contains(got, tt.wantContain) {
					t.Errorf("GetInstallPath() = %v, want to contain %v", got, tt.wantContain)
				}
				// Verify path starts with provided home directory
				if !strings.HasPrefix(got, tmpDir) {
					t.Errorf("GetInstallPath() = %v, expected path to start with %v", got, tmpDir)
				}
			}
		})
	}
}

func TestGetInstallPath_Zsh(t *testing.T) {
	tmpDir := t.TempDir()

	path, err := GetInstallPath(Zsh, tmpDir)
	if err != nil {
		t.Fatalf("GetInstallPath(Zsh) error = %v", err)
	}

	if !strings.Contains(path, ".zsh/completion/_pics") {
		t.Errorf("GetInstallPath(Zsh) = %v, want to contain .zsh/completion/_pics", path)
	}
}

func TestGetInstallPath_Fish(t *testing.T) {
	tmpDir := t.TempDir()

	path, err := GetInstallPath(Fish, tmpDir)
	if err != nil {
		t.Fatalf("GetInstallPath(Fish) error = %v", err)
	}

	if !strings.Contains(path, ".config/fish/completions/pics.fish") {
		t.Errorf("GetInstallPath(Fish) = %v, want to contain .config/fish/completions/pics.fish", path)
	}
}

