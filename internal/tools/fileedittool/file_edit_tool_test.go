package fileedittool_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

func TestFileEdit_replaceOne(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	st := stateFor(t, abs, "a\nb\nc\n")
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	fe := fileedittool.New()
	in, _ := json.Marshal(map[string]any{"file_path": abs, "old_string": "b", "new_string": "x", "replace_all": false})
	out, err := fe.Run(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "a\nx\nc\n" {
		t.Fatalf("%q", b)
	}
}

func TestFileEdit_replaceAll(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("x x"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	st := stateFor(t, abs, "x x")
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	fe := fileedittool.New()
	in, _ := json.Marshal(map[string]any{"file_path": abs, "old_string": "x", "new_string": "y", "replace_all": true})
	_, err := fe.Run(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "y y" {
		t.Fatalf("%q", b)
	}
}

func TestFileEdit_noReadState(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("ab"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fe := fileedittool.New()
	in, _ := json.Marshal(map[string]any{"file_path": abs, "old_string": "a", "new_string": "z", "replace_all": false})
	_, err := fe.Run(context.Background(), in)
	if err == nil || !strings.Contains(err.Error(), "not been read") {
		t.Fatalf("%v", err)
	}
}

func TestFileEdit_sameStrings(t *testing.T) {
	fe := fileedittool.New()
	in, _ := json.Marshal(map[string]any{"file_path": "/tmp/x", "old_string": "a", "new_string": "a", "replace_all": false})
	_, err := fe.Run(context.Background(), in)
	if err == nil || !strings.Contains(err.Error(), "No changes") {
		t.Fatalf("%v", err)
	}
}

func TestFileEdit_notFoundIncludesSuggestPathUnderCwd(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(repo, "missing_segment.go")
	if err := os.WriteFile(target, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wrong, err := filepath.Abs(filepath.Join(root, "missing_segment.go"))
	if err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	fe := fileedittool.New()
	in, _ := json.Marshal(map[string]any{"file_path": wrong, "old_string": "x", "new_string": "y", "replace_all": false})
	_, err = fe.Run(context.Background(), in)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "File does not exist") || !strings.Contains(msg, "Did you mean") {
		t.Fatalf("got %q", msg)
	}
	if !strings.Contains(msg, "missing_segment.go") {
		t.Fatalf("expected suggested filename in message: %q", msg)
	}
}

func TestFileEdit_createEmptyPath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "new.txt")
	abs, _ := filepath.Abs(p)
	fe := fileedittool.New()
	in, _ := json.Marshal(map[string]any{"file_path": abs, "old_string": "", "new_string": "hi", "replace_all": false})
	_, err := fe.Run(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "hi" {
		t.Fatalf("%q", b)
	}
}

func TestFileEdit_ipynbRejected(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.ipynb")
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	st := stateFor(t, abs, "{}")
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	fe := fileedittool.New()
	in, _ := json.Marshal(map[string]any{"file_path": abs, "old_string": "{", "new_string": "[", "replace_all": false})
	_, err := fe.Run(ctx, in)
	if err == nil || !strings.Contains(err.Error(), "Notebook") {
		t.Fatalf("%v", err)
	}
}

func TestMapEditToolResultForMessagesAPI(t *testing.T) {
	s := fileedittool.MapEditToolResultForMessagesAPI([]byte(`{"filePath":"/p","userModified":false,"replaceAll":false}`))
	if !strings.Contains(s, "successfully") {
		t.Fatal(s)
	}
	ra := fileedittool.MapEditToolResultForMessagesAPI([]byte(`{"filePath":"/p","userModified":false,"replaceAll":true}`))
	if !strings.Contains(ra, "All occurrences") {
		t.Fatal(ra)
	}
}

func TestFileEdit_implementsTool(t *testing.T) {
	var _ tools.Tool = fileedittool.New()
}

func stateFor(t *testing.T, abs, content string) *filereadtool.ReadFileStateMap {
	t.Helper()
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st := filereadtool.NewReadFileStateMap()
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       content,
		Timestamp:     fi.ModTime().UnixMilli(),
		IsPartialView: false,
	})
	return st
}
