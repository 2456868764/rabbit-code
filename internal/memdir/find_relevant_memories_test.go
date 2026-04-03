package memdir

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindRelevantMemories_heuristic(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "z.md"), []byte("zebra stripes"), 0o600)
	paths, err := FindRelevantMemories(context.Background(), "zebra", dir, FindRelevantMemoriesOpts{
		Mode:  RelevanceModeHeuristic,
		Limit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || filepath.Base(paths[0]) != "z.md" {
		t.Fatalf("%#v", paths)
	}
}

func TestFindRelevantMemories_llmUsesTextComplete(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "pick.md"), []byte("content"), 0o600)
	paths, err := FindRelevantMemories(context.Background(), "q", dir, FindRelevantMemoriesOpts{
		Mode: RelevanceModeLLM,
		TextComplete: func(context.Context, string, string) (string, error) {
			return `{"selected_memories":["pick.md"]}`, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || filepath.Base(paths[0]) != "pick.md" {
		t.Fatalf("%#v", paths)
	}
}

func TestFindRelevantMemories_llmEmptyJSONHonored(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "x.md"), []byte("zebra"), 0o600)
	paths, err := FindRelevantMemories(context.Background(), "zebra", dir, FindRelevantMemoriesOpts{
		Mode: RelevanceModeLLM,
		TextComplete: func(context.Context, string, string) (string, error) {
			return `{"selected_memories":[]}`, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected no LLM picks, got %#v", paths)
	}
}

func TestFindRelevantMemories_llmFallbackHeuristic(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("banana facts"), 0o600)
	paths, err := FindRelevantMemories(context.Background(), "banana", dir, FindRelevantMemoriesOpts{
		Mode: RelevanceModeLLM,
		TextComplete: func(context.Context, string, string) (string, error) {
			return "", os.ErrInvalid
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("%#v", paths)
	}
}

func TestFindRelevantMemories_alreadySurfaced(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "one.md")
	_ = os.WriteFile(p, []byte("banana"), 0o600)
	paths, err := FindRelevantMemories(context.Background(), "banana", dir, FindRelevantMemoriesOpts{
		Mode:            RelevanceModeHeuristic,
		AlreadySurfaced: map[string]struct{}{p: {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("%#v", paths)
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

func TestFindRelevantMemories_llmResolvesRelativePath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(sub, "deep.md"), []byte("x"), 0o600)
	rel := filepath.ToSlash("nested/deep.md")
	paths, err := FindRelevantMemories(context.Background(), "q", dir, FindRelevantMemoriesOpts{
		Mode: RelevanceModeLLM,
		TextComplete: func(context.Context, string, string) (string, error) {
			return `{"selected_memories":["` + rel + `"]}`, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || filepath.Base(paths[0]) != "deep.md" {
		t.Fatalf("%#v", paths)
	}
}

func TestFindRelevantMemories_strictLLM_noHeuristicFallback(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("banana facts"), 0o600)
	paths, err := FindRelevantMemories(context.Background(), "banana", dir, FindRelevantMemoriesOpts{
		Mode:      RelevanceModeLLM,
		StrictLLM: true,
		TextComplete: func(context.Context, string, string) (string, error) {
			return "", os.ErrInvalid
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("strict: want no paths, got %#v", paths)
	}
}

func TestFindRelevantMemoriesDetailed_carriesMtime(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.md")
	_ = os.WriteFile(p, []byte("x"), 0o600)
	got, err := FindRelevantMemoriesDetailed(context.Background(), "nope", dir, FindRelevantMemoriesOpts{
		Mode:  RelevanceModeHeuristic,
		Limit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		// no token overlap with "nope"
	}
	_ = os.WriteFile(filepath.Join(dir, "z.md"), []byte("nope match here"), 0o600)
	got, err = FindRelevantMemoriesDetailed(context.Background(), "nope", dir, FindRelevantMemoriesOpts{
		Mode:  RelevanceModeHeuristic,
		Limit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].MtimeMs == 0 || filepath.Base(got[0].Path) != "z.md" {
		t.Fatalf("%+v", got)
	}
}

func TestFindRelevantMemories_OnRecallShape(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("x"), 0o600)
	var cand, sel int
	_, err := FindRelevantMemories(context.Background(), "q", dir, FindRelevantMemoriesOpts{
		Mode: RelevanceModeLLM,
		TextComplete: func(context.Context, string, string) (string, error) {
			return `{"selected_memories":[]}`, nil
		},
		OnRecallShape: func(c, s int) { cand, sel = c, s },
	})
	if err != nil {
		t.Fatal(err)
	}
	if cand != 1 || sel != 0 {
		t.Fatalf("cand=%d sel=%d", cand, sel)
	}
}
