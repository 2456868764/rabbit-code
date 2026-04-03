package memdir

import (
	"strings"
	"testing"
	"time"
)

func TestMemoryAttachmentHeaderAt_fresh(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	m := now.UnixMilli()
	h := MemoryAttachmentHeaderAt("/tmp/a.md", m, now)
	if !strings.Contains(h, "today") || !strings.Contains(h, "/tmp/a.md") {
		t.Fatalf("%q", h)
	}
	if strings.Contains(h, "This memory is") {
		t.Fatal("unexpected staleness block")
	}
}

func TestMemoryAttachmentHeaderAt_stale(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	old := now.Add(-5 * 24 * time.Hour).UnixMilli()
	h := MemoryAttachmentHeaderAt("/x.md", old, now)
	if !strings.HasPrefix(h, "This memory is") || !strings.Contains(h, "Memory: /x.md:") {
		t.Fatalf("%q", h)
	}
}
