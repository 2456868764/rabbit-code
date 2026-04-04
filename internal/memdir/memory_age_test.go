package memdir

import (
	"strings"
	"testing"
	"time"
)

func TestMemoryAgeDaysAt_futureMtime(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour).UnixMilli()
	if MemoryAgeDaysAt(future, now) != 0 {
		t.Fatal()
	}
}

func TestMemoryAgeAt_phrases(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	if s := MemoryAgeAt(now.UnixMilli(), now); s != "today" {
		t.Fatalf("%q", s)
	}
	y := now.Add(-25 * time.Hour).UnixMilli()
	if s := MemoryAgeAt(y, now); s != "yesterday" {
		t.Fatalf("%q", s)
	}
	old := now.Add(-5 * 24 * time.Hour).UnixMilli()
	if s := MemoryAgeAt(old, now); s != "5 days ago" {
		t.Fatalf("%q", s)
	}
}

func TestMemoryFreshnessTextAt_threshold(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	today := now.UnixMilli()
	if MemoryFreshnessTextAt(today, now) != "" {
		t.Fatal()
	}
	y := now.Add(-30 * time.Hour).UnixMilli()
	if MemoryFreshnessTextAt(y, now) != "" {
		t.Fatal()
	}
	old := now.Add(-3 * 24 * time.Hour).UnixMilli()
	s := MemoryFreshnessTextAt(old, now)
	if s == "" || !strings.Contains(s, "3 days old") {
		t.Fatalf("%q", s)
	}
}

func TestMemoryFreshnessNoteAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	old := now.Add(-10 * 24 * time.Hour).UnixMilli()
	s := MemoryFreshnessNoteAt(old, now)
	if !strings.HasPrefix(s, "<system-reminder>") || !strings.Contains(s, "</system-reminder>") {
		t.Fatalf("%q", s)
	}
}

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
