package memdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestSessionFragmentsFromPaths(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(a, []byte("  hello  \n"), 0o600)
	_ = os.WriteFile(b, []byte("world"), 0o600)
	frags, raw, err := SessionFragmentsFromPaths([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if raw != len([]byte("  hello  \n"))+len([]byte("world")) {
		t.Fatalf("raw bytes %d", raw)
	}
	if len(frags) != 2 || frags[0] != "hello" || frags[1] != "world" {
		t.Fatalf("%#v", frags)
	}
}

func TestSessionFragmentsFromPaths_missing(t *testing.T) {
	_, _, err := SessionFragmentsFromPaths([]string{"/nonexistent/memdir.txt"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionFragmentsFromPathsWithAttachmentHeadersAt(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.md")
	_ = os.WriteFile(p, []byte("  body  "), 0o600)
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	frags, raw, err := SessionFragmentsFromPathsWithAttachmentHeadersAt([]string{p}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(frags) != 1 {
		t.Fatalf("%#v", frags)
	}
	if !strings.Contains(frags[0], "body") || !strings.Contains(frags[0], p) {
		t.Fatalf("%q", frags[0])
	}
	if raw != len(frags[0]) {
		t.Fatalf("raw %d len frag %d", raw, len(frags[0]))
	}
}

func TestFindGitRoot_findsAncestor(t *testing.T) {
	top := t.TempDir()
	if err := os.MkdirAll(filepath.Join(top, "a", "b"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(top, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	leaf := filepath.Join(top, "a", "b")
	got, ok := FindGitRoot(leaf)
	if !ok || got != top {
		t.Fatalf("got %q %v", got, ok)
	}
}

func TestFindGitRoot_none(t *testing.T) {
	dir := t.TempDir()
	_, ok := FindGitRoot(dir)
	if ok {
		t.Fatal("expected no git root")
	}
}

func TestSanitizePath_ascii(t *testing.T) {
	got := SanitizePath("/Users/foo/my-project")
	want := "-Users-foo-my-project"
	if got != want {
		t.Fatalf("%q", got)
	}
}

func TestSanitizePath_truncatesWithHash(t *testing.T) {
	long := strings.Repeat("a", MaxSanitizedLength+30)
	got := SanitizePath(long)
	if len(got) <= MaxSanitizedLength {
		t.Fatalf("expected hash suffix extension, got len %d", len(got))
	}
	if !strings.Contains(got, "-") {
		t.Fatal(got)
	}
}

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

func TestResolveAutoMemDir_trustedSettings(t *testing.T) {
	for _, k := range []string{
		"RABBIT_CODE_MEMORY_PATH_OVERRIDE",
		"CLAUDE_COWORK_MEMORY_PATH_OVERRIDE",
	} {
		t.Setenv(k, "")
	}
	mem := t.TempDir()
	absMem, err := filepath.Abs(mem)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(absMem) + string(filepath.Separator)
	got, err := ResolveAutoMemDirWithOptions(t.TempDir(), AutoMemResolveOptions{
		TrustedAutoMemoryDirectory: absMem,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveAutoMemDir_trustedTilde(t *testing.T) {
	for _, k := range []string{
		"RABBIT_CODE_MEMORY_PATH_OVERRIDE",
		"CLAUDE_COWORK_MEMORY_PATH_OVERRIDE",
	} {
		t.Setenv(k, "")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	sub := filepath.Join(home, "memdir", "x")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	absWant, err := filepath.Abs(sub)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(absWant) + string(filepath.Separator)
	got, err := ResolveAutoMemDirWithOptions(t.TempDir(), AutoMemResolveOptions{
		TrustedAutoMemoryDirectory: "~/memdir/x",
	})
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

func TestIsExtractModeActive_envParity(t *testing.T) {
	t.Setenv(features.EnvExtractMemories, "")
	t.Setenv(features.EnvExtractMemoriesNonInteractive, "")
	if IsExtractModeActive(false) {
		t.Fatal("extract env off should be inactive")
	}
	t.Setenv(features.EnvExtractMemories, "1")
	if !IsExtractModeActive(false) {
		t.Fatal("interactive + env on")
	}
	if IsExtractModeActive(true) {
		t.Fatal("non-interactive without override")
	}
	t.Setenv(features.EnvExtractMemoriesNonInteractive, "1")
	if !IsExtractModeActive(true) {
		t.Fatal("non-interactive with override")
	}
}
