package registry

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/2456868764/rabbit-code/internal/tools"
)

// Registry holds builtin tools and dynamically registered MCP tools (TS: mcp tools merged into getTools() list).
type Registry struct {
	mu      sync.RWMutex
	builtin []tools.Tool
	mcp     []tools.Tool
}

// New returns a registry with immutable builtin tools (copied). MCP list starts empty.
func New(builtin ...tools.Tool) *Registry {
	cp := make([]tools.Tool, len(builtin))
	copy(cp, builtin)
	return &Registry{builtin: cp}
}

// RegisterMCP appends a tool typically discovered from an MCP server. Returns error if primary name is already registered (builtin or MCP).
func (r *Registry) RegisterMCP(t tools.Tool) error {
	if t == nil {
		return fmt.Errorf("registry: RegisterMCP: nil tool")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.findLocked(t.Name()) != nil {
		return fmt.Errorf("registry: tool %q already registered", t.Name())
	}
	r.mcp = append(r.mcp, t)
	return nil
}

// UnregisterMCP removes an MCP tool by primary name. Returns false if not found.
func (r *Registry) UnregisterMCP(primaryName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, x := range r.mcp {
		if x.Name() == primaryName {
			r.mcp = append(r.mcp[:i], r.mcp[i+1:]...)
			return true
		}
	}
	return false
}

// ListNames returns sorted primary names for builtin tools then MCP tools (deduped by name).
func (r *Registry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	var out []string
	for _, x := range r.builtin {
		n := x.Name()
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	for _, x := range r.mcp {
		n := x.Name()
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ByName returns the tool matching primary name or alias, or nil.
func (r *Registry) ByName(name string) tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findLocked(name)
}

func (r *Registry) findLocked(name string) tools.Tool {
	if t := tools.Find(r.builtin, name); t != nil {
		return t
	}
	return tools.Find(r.mcp, name)
}

// RunTool implements query.ToolRunner: dispatches to ByName then Tool.Run.
func (r *Registry) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	t := r.ByName(name)
	if t == nil {
		return nil, fmt.Errorf("registry: unknown tool %q", name)
	}
	return t.Run(ctx, inputJSON)
}
