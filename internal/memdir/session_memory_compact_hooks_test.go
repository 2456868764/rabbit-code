package memdir

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionMemoryCompactHooksForMemoryDir_readsEntrypoint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, EntrypointName), []byte("# Title\n\nhello"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := SessionMemoryCompactHooksForMemoryDir(dir)
	if h.GetSessionMemoryContent == nil {
		t.Fatal("expected GetSessionMemoryContent")
	}
	s, err := h.GetSessionMemoryContent(context.Background())
	if err != nil || !strings.Contains(s, "hello") {
		t.Fatalf("content: %q err=%v", s, err)
	}
	empty, err := h.IsSessionMemoryEmpty(context.Background(), s)
	if err != nil || empty {
		t.Fatalf("empty=%v err=%v", empty, err)
	}
	if h.SessionMemoryPathForFooter == nil {
		t.Fatal("footer path")
	}
	if !strings.Contains(h.SessionMemoryPathForFooter(), EntrypointName) {
		t.Fatalf("footer %q", h.SessionMemoryPathForFooter())
	}
}

func TestSessionMemoryCompactHooksForMemoryDir_missingFile(t *testing.T) {
	dir := t.TempDir()
	h := SessionMemoryCompactHooksForMemoryDir(dir)
	s, err := h.GetSessionMemoryContent(context.Background())
	if err != nil || strings.TrimSpace(s) != "" {
		t.Fatalf("want empty, got %q err=%v", s, err)
	}
}
