package registry_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
	"github.com/2456868764/rabbit-code/internal/tools/registry"
)

type stubTool struct {
	name, alias string
	out         []byte
	err         error
}

func (s stubTool) Name() string      { return s.name }
func (s stubTool) Aliases() []string { return []string{s.alias} }
func (s stubTool) Run(context.Context, []byte) ([]byte, error) {
	return s.out, s.err
}

func TestRegistryImplementsQueryToolRunner(t *testing.T) {
	var _ query.ToolRunner = (*registry.Registry)(nil)
}

func TestMatchesName_viaRegistry(t *testing.T) {
	r := registry.New(stubTool{name: "Read", alias: "read"})
	if r.ByName("Read") == nil || r.ByName("read") == nil {
		t.Fatal("expected alias lookup")
	}
	if r.ByName("Write") != nil {
		t.Fatal("expected nil")
	}
}

func TestRegisterMCP_UnregisterMCP(t *testing.T) {
	r := registry.New(stubTool{name: "bash", out: []byte(`{"ok":true}`)})
	mcp := stubTool{name: "mcp__srv__ping", out: []byte(`{"pong":1}`)}
	if err := r.RegisterMCP(mcp); err != nil {
		t.Fatal(err)
	}
	names := r.ListNames()
	if len(names) != 2 {
		t.Fatalf("%v", names)
	}
	out, err := r.RunTool(context.Background(), "mcp__srv__ping", []byte("{}"))
	if err != nil || string(out) != `{"pong":1}` {
		t.Fatalf("%v %s", err, out)
	}
	if !r.UnregisterMCP("mcp__srv__ping") {
		t.Fatal("unregister")
	}
	if r.ByName("mcp__srv__ping") != nil {
		t.Fatal("still registered")
	}
	_, err = r.RunTool(context.Background(), "mcp__srv__ping", []byte("{}"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterMCP_duplicate(t *testing.T) {
	a := stubTool{name: "t1"}
	r := registry.New(a)
	if err := r.RegisterMCP(stubTool{name: "t1"}); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestToolsMatchesName(t *testing.T) {
	x := stubTool{name: "A", alias: "a"}
	if !tools.MatchesName(x, "A") || !tools.MatchesName(x, "a") || tools.MatchesName(x, "b") {
		t.Fatal()
	}
}

func TestRegistry_withFileReadTool(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(p, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	r := registry.New(filereadtool.New())
	in, _ := json.Marshal(map[string]string{"file_path": abs})
	out, err := r.RunTool(context.Background(), "Read", in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil || m["type"] != "text" {
		t.Fatalf("%v %s", err, out)
	}
}

func TestRegistry_withFileReadAndWrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rw.txt")
	if err := os.WriteFile(p, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	st := filereadtool.NewReadFileStateMap()
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       "a",
		Timestamp:     fi.ModTime().UnixMilli(),
		IsPartialView: false,
	})
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	r := registry.New(filereadtool.New(), filewritetool.New())
	win, _ := json.Marshal(map[string]string{"file_path": abs, "content": "b"})
	wout, err := r.RunTool(ctx, "Write", win)
	if err != nil {
		t.Fatal(err)
	}
	var wm map[string]any
	if err := json.Unmarshal(wout, &wm); err != nil || wm["type"] != "update" {
		t.Fatalf("%v %s", err, wout)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "b" {
		t.Fatalf("disk %q", b)
	}
}

func TestRegistry_withFileEditTool(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "e.txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	st := filereadtool.NewReadFileStateMap()
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       "hello world",
		Timestamp:     fi.ModTime().UnixMilli(),
		IsPartialView: false,
	})
	ctx := filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
	r := registry.New(filereadtool.New(), filewritetool.New(), fileedittool.New())
	in, _ := json.Marshal(map[string]any{
		"file_path":   abs,
		"old_string":  "world",
		"new_string":  "go",
		"replace_all": false,
	})
	out, err := r.RunTool(ctx, "Edit", in)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["newString"] != "go" {
		t.Fatalf("%v", m)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "hello go" {
		t.Fatalf("disk %q", b)
	}
}
