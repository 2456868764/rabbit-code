package filewritetool_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
)

func TestFileWrite_NewFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "new.txt")
	abs, _ := filepath.Abs(p)
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "hello"})
	out, err := fw.Run(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil || m["type"] != "create" {
		t.Fatalf("%v %s", err, out)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "hello" {
		t.Fatalf("disk %q", b)
	}
}

func TestFileWrite_UpdateWithoutReadState(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "new"})
	_, err := fw.Run(context.Background(), in)
	if err == nil {
		t.Fatal("expected error without read state")
	}
}

func TestFileWrite_UpdateAfterReadState(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	st := filereadtool.NewReadFileStateMap()
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       "old",
		Timestamp:     fi.ModTime().UnixMilli(),
		IsPartialView: false,
	})
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "new"})
	out, err := fw.Run(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil || m["type"] != "update" {
		t.Fatalf("%v %s", err, out)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "new" {
		t.Fatal(string(b))
	}
}

func TestFileWrite_DenyEdit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	abs, _ := filepath.Abs(p)
	ctx := filewritetool.WithWriteContext(context.Background(), &filewritetool.WriteContext{
		DenyEdit: func(string) bool { return true },
	})
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "x"})
	_, err := fw.Run(ctx, in)
	if err == nil {
		t.Fatal("expected deny")
	}
}

func TestFileWrite_badJSON(t *testing.T) {
	fw := filewritetool.New()
	_, err := fw.Run(context.Background(), []byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileWrite_strictJSONUnknownField(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "n.txt")
	abs, _ := filepath.Abs(p)
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]any{"file_path": abs, "content": "x", "extra": 1})
	_, err := fw.Run(context.Background(), in)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("got %v", err)
	}
}

func TestFileWrite_modifiedSinceRead_validateStrictMtime(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	st := filereadtool.NewReadFileStateMap()
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       "old",
		Timestamp:     0,
		IsPartialView: false,
	})
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "new"})
	_, err := fw.Run(ctx, in)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "modified since read") {
		t.Fatalf("got %v", err)
	}
}

func TestFileWrite_checkTeamMemSecrets(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "n.txt")
	abs, _ := filepath.Abs(p)
	ctx := filewritetool.WithWriteContext(context.Background(), &filewritetool.WriteContext{
		CheckTeamMemSecrets: func(_, _ string) string {
			return "blocked"
		},
	})
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "x"})
	_, err := fw.Run(ctx, in)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("%v", err)
	}
}

func TestFileWrite_gitDiffOptional(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REMOTE", "true")
	dir := t.TempDir()
	p := filepath.Join(dir, "gitdiff.txt")
	abs, _ := filepath.Abs(p)
	ctx := filewritetool.WithWriteContext(context.Background(), &filewritetool.WriteContext{
		QuartzLanternEnabled: func() bool { return true },
		FetchGitDiff: func(string) (map[string]any, error) {
			return map[string]any{"filename": "gitdiff.txt", "status": "added"}, nil
		},
	})
	fw := filewritetool.New()
	in, _ := json.Marshal(map[string]string{"file_path": abs, "content": "hello"})
	out, err := fw.Run(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["gitDiff"] == nil {
		t.Fatalf("missing gitDiff: %s", out)
	}
}

func TestFileWrite_implementsTool(t *testing.T) {
	var _ tools.Tool = filewritetool.New()
}

func TestMapWriteToolResultForMessagesAPI(t *testing.T) {
	c := filewritetool.MapWriteToolResultForMessagesAPI([]byte(`{"type":"create","filePath":"/tmp/x"}`))
	if !strings.Contains(c, "created") || !strings.Contains(c, "/tmp/x") {
		t.Fatalf("%q", c)
	}
	u := filewritetool.MapWriteToolResultForMessagesAPI([]byte(`{"type":"update","filePath":"/tmp/y"}`))
	if !strings.Contains(u, "updated") {
		t.Fatalf("%q", u)
	}
}
