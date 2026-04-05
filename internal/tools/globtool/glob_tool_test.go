package globtool_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
	"github.com/2456868764/rabbit-code/internal/tools/registry"
)

func requireRg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep (rg) not in PATH")
	}
}

func TestGlob_implementsTool(t *testing.T) {
	var _ tools.Tool = globtool.New()
}

func TestGlob_MapGlobToolResultForMessagesAPI(t *testing.T) {
	s := globtool.MapGlobToolResultForMessagesAPI([]byte(`{"filenames":["a","b"],"truncated":false}`))
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Fatal(s)
	}
	if globtool.MapGlobToolResultForMessagesAPI([]byte(`{"filenames":[],"truncated":false}`)) != "No files found" {
		t.Fatal()
	}
}

func TestGlob_strictJSON(t *testing.T) {
	g := globtool.New()
	_, err := g.Run(context.Background(), []byte(`{"pattern":"*.go","extra":1}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("got %v", err)
	}
}

func TestGlob_matchesTxt(t *testing.T) {
	requireRg(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := globtool.New()
	in, _ := json.Marshal(map[string]string{"pattern": "*.txt", "path": dir})
	out, err := g.Run(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if int(m["numFiles"].(float64)) != 1 {
		t.Fatalf("%v", m)
	}
	fns, _ := m["filenames"].([]any)
	if len(fns) != 1 || !strings.HasSuffix(fns[0].(string), "a.txt") {
		t.Fatalf("%v", m)
	}
}

func TestGlob_missingDir(t *testing.T) {
	requireRg(t)
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope")
	g := globtool.New()
	in, _ := json.Marshal(map[string]string{"pattern": "*", "path": missing})
	_, err := g.Run(context.Background(), in)
	if err == nil || !strings.Contains(err.Error(), "Directory does not exist") {
		t.Fatalf("%v", err)
	}
}

func TestGlob_notDirectory(t *testing.T) {
	requireRg(t)
	f := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := globtool.New()
	in, _ := json.Marshal(map[string]string{"pattern": "*", "path": f})
	_, err := g.Run(context.Background(), in)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("%v", err)
	}
}

func TestRegistry_withGlobTool(t *testing.T) {
	requireRg(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "z.txt"), []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := registry.New(globtool.New())
	in, _ := json.Marshal(map[string]string{"pattern": "*.txt", "path": dir})
	out, err := r.RunTool(context.Background(), "Glob", in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["numFiles"].(float64) < 1 {
		t.Fatalf("%v", m)
	}
}
