package fileedittool

import (
	"os"
	"path/filepath"
	"strings"
)

// SuggestPathUnderCwd mirrors utils/file.ts suggestPathUnderCwd (sync, EvalSymlinks on parent dir).
func SuggestPathUnderCwd(requestedAbs string) (corrected string, ok bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	cwd = filepath.Clean(cwd)
	cwdParent := filepath.Dir(cwd)
	sep := string(filepath.Separator)

	reqDir := filepath.Dir(requestedAbs)
	baseName := filepath.Base(requestedAbs)

	resolvedDir := filepath.Clean(reqDir)
	if _, err := os.Stat(reqDir); err == nil {
		if ev, err := filepath.EvalSymlinks(reqDir); err == nil {
			resolvedDir = filepath.Clean(ev)
		}
	}
	resolvedPath := filepath.Clean(filepath.Join(resolvedDir, baseName))

	var cwdParentPrefix string
	if cwdParent == sep {
		cwdParentPrefix = sep
	} else {
		cwdParentPrefix = cwdParent + sep
	}

	if cwdParent != sep {
		if !strings.HasPrefix(resolvedPath, cwdParentPrefix) {
			return "", false
		}
	} else {
		if !strings.HasPrefix(resolvedPath, sep) {
			return "", false
		}
	}

	if resolvedPath == cwd || strings.HasPrefix(resolvedPath, cwd+sep) {
		return "", false
	}

	relFromParent, err := filepath.Rel(cwdParent, resolvedPath)
	if err != nil || relFromParent == ".." || strings.HasPrefix(relFromParent, ".."+sep) {
		return "", false
	}

	candidate := filepath.Clean(filepath.Join(cwd, relFromParent))
	if _, err := os.Stat(candidate); err != nil {
		return "", false
	}
	return candidate, true
}
