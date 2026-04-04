package memdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
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

func TestEnsureMemoryDirExists(t *testing.T) {
	t.Parallel()
	if err := EnsureMemoryDirExists(""); err != nil {
		t.Fatalf("empty: %v", err)
	}
	base := t.TempDir()
	sub := filepath.Join(base, "nested", "mem")
	if err := EnsureMemoryDirExists(sub); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsDir() {
		t.Fatalf("want dir, got %v", st.Mode())
	}
}

func TestLoadMemorySystemPrompt_gates(t *testing.T) {
	t.Setenv(features.EnvDisableAutoMemory, "")
	t.Setenv(features.EnvMemorySystemPrompt, "0")
	_, ok := LoadMemorySystemPrompt(MemorySystemPromptInput{
		MemoryDir: "/tmp/mem",
		Merged:    map[string]interface{}{"autoMemoryEnabled": true},
	})
	if ok {
		t.Fatal("expected off when MEMORY_SYSTEM_PROMPT falsy")
	}
}

func TestLoadMemorySystemPrompt_autoOnlyShape(t *testing.T) {
	t.Setenv(features.EnvDisableAutoMemory, "")
	t.Setenv(features.EnvMemorySystemPrompt, "1")
	t.Setenv(features.EnvTeamMem, "")
	t.Setenv(features.EnvKairosDailyLogMemory, "")
	t.Setenv(features.EnvKairosActive, "")
	t.Setenv(features.EnvMemorySearchPastContext, "")
	dir := t.TempDir()
	s, ok := LoadMemorySystemPrompt(MemorySystemPromptInput{
		MemoryDir:   dir,
		ProjectRoot: dir,
		Merged:      map[string]interface{}{"autoMemoryEnabled": true},
	})
	if !ok || !strings.Contains(s, "# auto memory") {
		t.Fatalf("ok=%v head=%q", ok, truncateTestString(s, 80))
	}
}

func TestBuildSearchingPastContextSection_gate(t *testing.T) {
	t.Setenv(features.EnvMemorySearchPastContext, "")
	if len(BuildSearchingPastContextSection("/mem/", "/proj", false)) != 0 {
		t.Fatal("expected empty when env off")
	}
	t.Setenv(features.EnvMemorySearchPastContext, "1")
	lines := BuildSearchingPastContextSection("/mem/", "/proj", false)
	if len(lines) == 0 || !strings.Contains(strings.Join(lines, "\n"), ".jsonl") {
		t.Fatalf("got %v", lines)
	}
}

func TestSessionFragments_nilByDefault(t *testing.T) {
	if s := SessionFragments(); s != nil {
		t.Fatalf("want nil slice, got %#v", s)
	}
}

func TestBuildMemoryPrompt_inlinesEntrypoint(t *testing.T) {
	t.Setenv(features.EnvMemorySearchPastContext, "")
	dir := t.TempDir()
	memFile := filepath.Join(dir, EntrypointName)
	if err := os.WriteFile(memFile, []byte("index body"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := BuildMemoryPrompt(BuildMemoryPromptInput{
		DisplayName:     "agent memory",
		MemoryDir:       dir,
		ProjectRoot:     dir,
		ExtraGuidelines: nil,
	})
	if !strings.Contains(out, "# agent memory") || !strings.Contains(out, "## "+EntrypointName) || !strings.Contains(out, "index body") {
		t.Fatalf("unexpected prompt: %s", truncateTestString(out, 200))
	}
}

func TestBuildMemoryPrompt_emptyEntrypointMessage(t *testing.T) {
	t.Setenv(features.EnvMemorySearchPastContext, "")
	dir := t.TempDir()
	out := BuildMemoryPrompt(BuildMemoryPromptInput{
		DisplayName: "agent memory",
		MemoryDir:   dir,
		ProjectRoot: dir,
	})
	if !strings.Contains(out, "currently empty") {
		t.Fatalf("%s", truncateTestString(out, 160))
	}
}

func truncateTestString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
