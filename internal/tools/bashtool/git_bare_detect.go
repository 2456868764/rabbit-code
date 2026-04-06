package bashtool

import (
	"os"
	"path/filepath"
)

// IsBareGitExploitLayout mirrors utils/git.ts isCurrentDirectoryBareGitRepo for cwd `dir`:
// true when cwd looks like a bare repo or broken .git (HEAD/objects/refs at root) without a normal .git/HEAD file.
func IsBareGitExploitLayout(dir string) bool {
	cwd := filepath.Clean(dir)
	gitPath := filepath.Join(cwd, ".git")
	if st, err := os.Stat(gitPath); err == nil {
		if st.Mode().IsRegular() {
			return false
		}
		if st.IsDir() {
			headPath := filepath.Join(gitPath, "HEAD")
			if h, err := os.Stat(headPath); err == nil && h.Mode().IsRegular() {
				return false
			}
		}
	}
	if st, err := os.Stat(filepath.Join(cwd, "HEAD")); err == nil && st.Mode().IsRegular() {
		return true
	}
	if st, err := os.Stat(filepath.Join(cwd, "objects")); err == nil && st.IsDir() {
		return true
	}
	if st, err := os.Stat(filepath.Join(cwd, "refs")); err == nil && st.IsDir() {
		return true
	}
	return false
}
