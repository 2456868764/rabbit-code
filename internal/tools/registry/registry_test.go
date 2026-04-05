package registry_test

import (
	"context"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/tools"
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
