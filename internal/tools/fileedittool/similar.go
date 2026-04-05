package fileedittool

import (
	"os"
	"path/filepath"
	"strings"
)

// FindSimilarFile mirrors utils/file.ts findSimilarFile (same basename, different extension).
func FindSimilarFile(targetPath string) string {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		nex := filepath.Ext(n)
		nstem := strings.TrimSuffix(n, nex)
		if nstem == stem && filepath.Join(dir, n) != filepath.Clean(targetPath) {
			return n
		}
	}
	return ""
}
