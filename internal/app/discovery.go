package app

import (
	"os"
	"path/filepath"
	"runtime"
)

const configDirName = "rabbit-code"

// UserHome returns the current user's home directory (OS-specific).
func UserHome() (string, error) {
	return os.UserHomeDir()
}

// GlobalConfigDir returns the OS-appropriate user config directory for rabbit-code.
// Windows: os.UserConfigDir()/rabbit-code (%AppData% on typical setups).
// Unix: XDG_CONFIG_HOME/rabbit-code or ~/.config/rabbit-code.
func GlobalConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		cfg, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(cfg, configDirName), nil
	}
	home, err := UserHome()
	if err != nil {
		return "", err
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, configDirName), nil
	}
	return filepath.Join(home, ".config", configDirName), nil
}

// FindProjectRoot walks upward from startDir for markers: go.mod, .git, CLAUDE.md.
// Falls back to startDir if none found (or startDir is empty).
func FindProjectRoot(startDir string) string {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	dir := startDir
	for {
		if hasProjectMarker(dir, "go.mod") || hasProjectMarker(dir, "CLAUDE.md") || hasGitDir(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return startDir
		}
		dir = parent
	}
}

// ProjectConfigCandidate returns a conventional project-level config path (Phase 2 will parse).
func ProjectConfigCandidate(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}
	return filepath.Join(projectRoot, ".rabbit-code.json")
}

// DataDir returns the OS-appropriate user data directory for rabbit-code.
// Windows: os.UserCacheDir()/rabbit-code (typically under %LocalAppData%).
// macOS: ~/Library/Application Support/rabbit-code.
// Other Unix: XDG_DATA_HOME/rabbit-code or ~/.local/share/rabbit-code.
func DataDir() (string, error) {
	if runtime.GOOS == "windows" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, configDirName), nil
	}
	home, err := UserHome()
	if err != nil {
		return "", err
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, configDirName), nil
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", configDirName), nil
	}
	return filepath.Join(home, ".local", "share", configDirName), nil
}

func hasProjectMarker(dir, name string) bool {
	st, err := os.Stat(filepath.Join(dir, name))
	return err == nil && !st.IsDir()
}

func hasGitDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil // file (worktree) or directory
}
