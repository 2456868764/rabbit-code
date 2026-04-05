package memdir

// Corresponds to restored-src/src/services/compact/sessionMemoryCompact.ts for the file-backed hook surface only:
// GetSessionMemoryContent / empty check / path footer wired to auto-mem MEMORY.md (full compaction logic lives in internal/services/compact).

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// SessionMemoryCompactHooksForMemoryDir builds compact.SessionMemoryCompactHooks that read the session-memory
// entrypoint (MEMORY.md) under the resolved auto-memory directory, matching rabbit-code file-based memory.
// memoryDir should be the directory root (with or without trailing separator), same shape as engine memdirMemoryDir.
func SessionMemoryCompactHooksForMemoryDir(memoryDir string) compact.SessionMemoryCompactHooks {
	root := strings.TrimSpace(memoryDir)
	root = strings.TrimSuffix(root, string(filepath.Separator))
	if root == "" {
		return compact.SessionMemoryCompactHooks{}
	}
	entry := filepath.Join(root, EntrypointName)
	absEntry := entry
	if ap, err := filepath.Abs(entry); err == nil {
		absEntry = ap
	}
	return compact.SessionMemoryCompactHooks{
		GetSessionMemoryContent: func(ctx context.Context) (string, error) {
			_ = ctx
			b, err := os.ReadFile(entry)
			if err != nil {
				if os.IsNotExist(err) {
					return "", nil
				}
				return "", err
			}
			return string(b), nil
		},
		IsSessionMemoryEmpty: func(ctx context.Context, content string) (bool, error) {
			_ = ctx
			return strings.TrimSpace(content) == "", nil
		},
		SessionMemoryPathForFooter: func() string {
			return absEntry
		},
	}
}
