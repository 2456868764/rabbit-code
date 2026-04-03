package memdir

import (
	"strings"
	"testing"
)

func TestTruncateEntrypointContent_noTruncation(t *testing.T) {
	raw := "hello\nworld"
	got := TruncateEntrypointContent(raw)
	if got.WasLineTruncated || got.WasByteTruncated {
		t.Fatal()
	}
	if got.Content != raw {
		t.Fatalf("%q", got.Content)
	}
}

func TestTruncateEntrypointContent_lineCap(t *testing.T) {
	var lines []string
	for i := 0; i < MaxEntrypointLines+5; i++ {
		lines = append(lines, "x")
	}
	raw := strings.Join(lines, "\n")
	got := TruncateEntrypointContent(raw)
	if !got.WasLineTruncated {
		t.Fatal("expected line truncation")
	}
	before, _, ok := strings.Cut(got.Content, "\n\n> WARNING")
	if !ok {
		t.Fatal("missing warning block")
	}
	lineCount := strings.Count(before, "\n") + 1
	if lineCount != MaxEntrypointLines {
		t.Fatalf("want %d lines before warning, got %d", MaxEntrypointLines, lineCount)
	}
	if !strings.Contains(got.Content, "WARNING") {
		t.Fatal("missing warning")
	}
}

func TestTruncateEntrypointContent_byteCapLongLines(t *testing.T) {
	// Few lines but huge bytes — triggers byte-only path
	line := strings.Repeat("a", 15_000)
	raw := line + "\n" + line
	got := TruncateEntrypointContent(raw)
	if !got.WasByteTruncated {
		t.Fatalf("want byte trunc, line=%v byte=%v", got.WasLineTruncated, got.WasByteTruncated)
	}
	if len(got.Content) > MaxEntrypointBytes+4000 {
		t.Fatalf("content too long: %d", len(got.Content))
	}
}
