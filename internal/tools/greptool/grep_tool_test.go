package greptool_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
	"github.com/2456868764/rabbit-code/internal/tools/registry"
)

func TestGrep_implementsTool(t *testing.T) {
	var _ tools.Tool = greptool.New()
}

func TestMapGrepToolResultForMessagesAPI_filesMode(t *testing.T) {
	s := greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"files_with_matches","numFiles":2,"filenames":["a.go","b.go"]}`))
	if !strings.Contains(s, "Found 2 files") || !strings.Contains(s, "a.go") {
		t.Fatalf("unexpected: %q", s)
	}
	if greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"files_with_matches","numFiles":0,"filenames":[]}`)) != "No files found" {
		t.Fatal("empty files mode")
	}
}

func TestMapGrepToolResultForMessagesAPI_contentMode(t *testing.T) {
	s := greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"content","numFiles":0,"filenames":[],"content":"x:1:hi"}`))
	if s != "x:1:hi" {
		t.Fatalf("got %q", s)
	}
	s2 := greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"content","content":"a","appliedLimit":10,"appliedOffset":5}`))
	if !strings.Contains(s2, "pagination = limit: 10, offset: 5") {
		t.Fatalf("want pagination in content map: %q", s2)
	}
}

func TestMapGrepToolResultForMessagesAPI_countMode(t *testing.T) {
	s := greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"count","numFiles":2,"numMatches":5,"content":"a:1\nb:4"}`))
	if !strings.Contains(s, "a:1") || !strings.Contains(s, "Found 5 total occurrences across 2 files.") {
		t.Fatalf("got %q", s)
	}
	s1 := greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"count","numFiles":1,"numMatches":1,"content":"x:1"}`))
	if !strings.Contains(s1, "1 total occurrence") || !strings.Contains(s1, "1 file.") {
		t.Fatalf("got %q", s1)
	}
	s3 := greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"count","numFiles":1,"numMatches":2,"content":"x:2","appliedLimit":3}`))
	if !strings.Contains(s3, "with pagination = limit: 3") {
		t.Fatalf("got %q", s3)
	}
}

func TestMapGrepToolResultForMessagesAPI_filesMode_singleFilePlural(t *testing.T) {
	s := greptool.MapGrepToolResultForMessagesAPI([]byte(`{"mode":"files_with_matches","numFiles":1,"filenames":["only.go"]}`))
	if !strings.Contains(s, "Found 1 file\n") {
		t.Fatalf("got %q", s)
	}
}

func TestGrep_missingPattern(t *testing.T) {
	g := greptool.New()
	_, err := g.Run(context.Background(), []byte(`{"pattern":"   "}`))
	if err == nil || !strings.Contains(err.Error(), "missing pattern") {
		t.Fatalf("got %v", err)
	}
}

func TestGrep_invalidOutputMode(t *testing.T) {
	g := greptool.New()
	_, err := g.Run(context.Background(), []byte(`{"pattern":"x","output_mode":"bogus"}`))
	if err == nil || !strings.Contains(err.Error(), "invalid output_mode") {
		t.Fatalf("got %v", err)
	}
}

func TestGrep_strictJSON(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("no ripgrep")
	}
	g := greptool.New()
	_, err := g.Run(context.Background(), []byte(`{"pattern":"x","extra":1}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestGrep_filesWithMatches_NODE_ENV_test_sortsByPath(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("no ripgrep")
	}
	t.Setenv("NODE_ENV", "test")
	dir := t.TempDir()
	for _, name := range []string{"z.txt", "a.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("needle\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := greptool.New()
	out, err := g.Run(context.Background(), []byte(`{"pattern":"needle","path":`+mustJSON(t, dir)+`}`))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Filenames []string `json:"filenames"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if len(o.Filenames) != 2 {
		t.Fatalf("got %v", o.Filenames)
	}
	if !strings.Contains(o.Filenames[0], "a.txt") || !strings.Contains(o.Filenames[1], "z.txt") {
		t.Fatalf("want lexicographic order a then z, got %v", o.Filenames)
	}
}

func TestGrep_filesWithMatches(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("no ripgrep")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "needle.txt"), []byte("alpha beta gamma\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := greptool.New()
	out, err := g.Run(context.Background(), []byte(`{"pattern":"beta","path":`+mustJSON(t, dir)+`}`))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Mode      string   `json:"mode"`
		NumFiles  int      `json:"numFiles"`
		Filenames []string `json:"filenames"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.NumFiles != 1 || len(o.Filenames) != 1 {
		t.Fatalf("got %+v raw %s", o, string(out))
	}
	if !strings.Contains(o.Filenames[0], "needle.txt") {
		t.Fatalf("filename %q", o.Filenames[0])
	}
}

func TestGrep_contentMode(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("no ripgrep")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("func Foo() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := greptool.New()
	out, err := g.Run(context.Background(), []byte(`{"pattern":"func","path":`+mustJSON(t, dir)+`,"output_mode":"content"}`))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Mode    string `json:"mode"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.Mode != "content" || !strings.Contains(o.Content, "func") {
		t.Fatalf("got %+v", o)
	}
}

func TestGrep_countMode(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("no ripgrep")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("a\na\na\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := greptool.New()
	out, err := g.Run(context.Background(), []byte(`{"pattern":"a","path":`+mustJSON(t, dir)+`,"output_mode":"count"}`))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Mode       string `json:"mode"`
		NumMatches int    `json:"numMatches"`
		NumFiles   int    `json:"numFiles"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.Mode != "count" || o.NumFiles != 1 || o.NumMatches != 3 {
		t.Fatalf("got %+v raw %s", o, string(out))
	}
}

func TestRegistry_withGrepTool(t *testing.T) {
	r := registry.New(greptool.New())
	if r.ByName("Grep") == nil {
		t.Fatal("expected Grep")
	}
}

func mustJSON(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
