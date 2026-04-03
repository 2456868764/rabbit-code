package memdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveAutoMemDir_override(t *testing.T) {
	tmp := t.TempDir()
	ov := filepath.Join(tmp, "memoverride") + string(filepath.Separator)
	if err := os.MkdirAll(ov, 0o700); err != nil {
		t.Fatal(err)
	}
	absOv, err := filepath.Abs(ov)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(absOv) + string(filepath.Separator)
	t.Setenv("RABBIT_CODE_MEMORY_PATH_OVERRIDE", absOv)
	t.Setenv("CLAUDE_COWORK_MEMORY_PATH_OVERRIDE", "")

	got, err := ResolveAutoMemDir("/irrelevant/root")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveAutoMemDir_projectsLayout(t *testing.T) {
	for _, k := range []string{
		"RABBIT_CODE_MEMORY_PATH_OVERRIDE",
		"CLAUDE_COWORK_MEMORY_PATH_OVERRIDE",
		"RABBIT_CODE_REMOTE_MEMORY_DIR",
		"CLAUDE_CODE_REMOTE_MEMORY_DIR",
	} {
		t.Setenv(k, "")
	}
	cfg := t.TempDir()
	t.Setenv("RABBIT_CODE_CONFIG_DIR", cfg)

	proj := t.TempDir()
	absProj, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveAutoMemDir(proj)
	if err != nil {
		t.Fatal(err)
	}
	seg := SanitizePath(absProj)
	want := filepath.Clean(filepath.Join(cfg, "projects", seg, "memory")) + string(filepath.Separator)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAutoMemDailyLogPath_shape(t *testing.T) {
	tmp := t.TempDir()
	ov := filepath.Join(tmp, "m") + string(filepath.Separator)
	_ = os.MkdirAll(ov, 0o700)
	absOv, _ := filepath.Abs(ov)
	t.Setenv("RABBIT_CODE_MEMORY_PATH_OVERRIDE", absOv)
	ts := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	p, err := AutoMemDailyLogPath("/x", ts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, filepath.Join("logs", "2026", "03", "2026-03-05.md")) {
		t.Fatalf("%q", p)
	}
}

func TestIsAutoMemPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memory") + string(filepath.Separator)
	_ = os.MkdirAll(root, 0o700)
	absRoot, _ := filepath.Abs(root)
	child := filepath.Join(strings.TrimSuffix(absRoot, string(filepath.Separator)), "a", "b.md")
	_ = os.MkdirAll(filepath.Dir(child), 0o700)
	if !IsAutoMemPath(child, absRoot) {
		t.Fatal("expected inside")
	}
	outside := filepath.Join(t.TempDir(), "other.txt")
	if IsAutoMemPath(outside, absRoot) {
		t.Fatal("expected outside")
	}
}

func TestHasAutoMemPathOverride(t *testing.T) {
	t.Setenv("RABBIT_CODE_MEMORY_PATH_OVERRIDE", "")
	if HasAutoMemPathOverride() {
		t.Fatal()
	}
	tmp := t.TempDir()
	abs, _ := filepath.Abs(tmp)
	t.Setenv("RABBIT_CODE_MEMORY_PATH_OVERRIDE", abs+string(filepath.Separator))
	if !HasAutoMemPathOverride() {
		t.Fatal()
	}
}
