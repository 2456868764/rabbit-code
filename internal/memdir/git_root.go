package memdir

import (
	"os"
	"path/filepath"
)

// FindGitRoot returns the absolute directory containing .git (paths.ts getAutoMemBase analogue).
// If start is a file path, search begins at its parent. Returns ("", false) if no ancestor has .git.
func FindGitRoot(start string) (abs string, ok bool) {
	start, err := filepath.Abs(filepath.Clean(start))
	if err != nil {
		return "", false
	}
	fi, err := os.Stat(start)
	if err == nil && !fi.IsDir() {
		start = filepath.Dir(start)
	}
	dir := start
	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}
