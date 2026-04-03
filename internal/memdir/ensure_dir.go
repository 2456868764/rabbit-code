package memdir

import (
	"os"
	"strings"
)

// EnsureMemoryDirExists creates memoryDir recursively (memdir.ts ensureMemoryDirExists); ignores EEXIST.
func EnsureMemoryDirExists(memoryDir string) error {
	memoryDir = strings.TrimSpace(memoryDir)
	if memoryDir == "" {
		return nil
	}
	return os.MkdirAll(memoryDir, 0o700)
}
