package memdir

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScanMemoryFiles_skipsMemoryMd_andNonMd(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("x"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("root"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "x.txt"), []byte("y"), 0o600)
	got, err := ScanMemoryFiles(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Filename != "a.md" {
		t.Fatalf("%+v", got)
	}
}

func TestScanMemoryFiles_frontmatterMeta(t *testing.T) {
	dir := t.TempDir()
	body := "---\ndescription: My desc\ntype: project\n---\nbody\n"
	_ = os.WriteFile(filepath.Join(dir, "f.md"), []byte(body), 0o600)
	got, err := ScanMemoryFiles(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Description != "My desc" || got[0].Type != "project" {
		t.Fatalf("%+v", got)
	}
}

func TestScanMemoryFiles_recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(sub, "inner.md"), []byte("deep"), 0o600)
	got, err := ScanMemoryFiles(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Filename != filepath.ToSlash("nested/inner.md") {
		t.Fatalf("%+v", got)
	}
}

func TestScanMemoryFiles_contextCancel(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 40; i++ {
		_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.md", i)), []byte("x"), 0o600)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := ScanMemoryFiles(ctx, dir)
	if err != nil {
		t.Fatalf("scanMemoryFiles.ts swallows errors: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("cancelled scan should yield empty list like TS catch, got %d", len(got))
	}
}

func TestScanMemoryFiles_missingDirReturnsEmpty(t *testing.T) {
	got, err := ScanMemoryFiles(context.Background(), filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if got != nil && len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestScanMemoryFiles_sortedNewestFirst(t *testing.T) {
	dir := t.TempDir()
	pOld := filepath.Join(dir, "old.md")
	pNew := filepath.Join(dir, "new.md")
	oldT := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newT := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = os.WriteFile(pOld, []byte("a"), 0o600)
	_ = os.WriteFile(pNew, []byte("b"), 0o600)
	if err := os.Chtimes(pOld, oldT, oldT); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(pNew, newT, newT); err != nil {
		t.Fatal(err)
	}
	got, err := ScanMemoryFiles(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Filename != "new.md" {
		t.Fatalf("order %+v", got)
	}
}

func TestFormatMemoryManifest(t *testing.T) {
	s := FormatMemoryManifest([]MemoryHeader{
		{Filename: "a.md", MtimeMs: 0, Description: "d1", Type: "reference"},
	})
	if s == "" || !strings.Contains(s, "a.md") || !strings.Contains(s, "d1") || !strings.Contains(s, "[reference]") {
		t.Fatalf("%q", s)
	}
}

// TS formatMemoryManifest appends ": description" only when description is truthy; null ↔ Description == "".
func TestFormatMemoryManifest_noDescriptionLikeTSNull(t *testing.T) {
	s := FormatMemoryManifest([]MemoryHeader{
		{Filename: "bare.md", MtimeMs: 1_700_000_000_000, Description: "", Type: ""},
	})
	if strings.Contains(s, "): ") {
		t.Fatalf("expected no description suffix after timestamp (TS null), got %q", s)
	}
	if !strings.Contains(s, "bare.md") {
		t.Fatalf("%q", s)
	}
}
