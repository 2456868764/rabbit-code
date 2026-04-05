package query

import (
	"context"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
	"github.com/2456868764/rabbit-code/internal/tools/notebookedittool"
	"github.com/2456868764/rabbit-code/internal/tools/registry"
	"github.com/2456868764/rabbit-code/internal/tools/todowritetool"
	"github.com/2456868764/rabbit-code/internal/tools/toolsearchtool"
	"github.com/2456868764/rabbit-code/internal/tools/webfetchtool"
	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
)

// NewDefaultToolRunner returns a ToolRunner with Phase-6 builtins (Read, Write, Edit, Glob, Grep, NotebookEdit, TodoWrite, WebFetch, WebSearch)
// and ToolSearch when features.ToolSearchEnabledOptimistic() matches upstream (utils/toolSearch.ts),
// plus BashExecToolRunner for tool name "bash" when not handled by the registry.
func NewDefaultToolRunner() ToolRunner {
	builtins := []tools.Tool{
		filereadtool.New(),
		filewritetool.New(),
		fileedittool.New(),
		globtool.New(),
		greptool.New(),
		notebookedittool.New(),
		todowritetool.New(),
		webfetchtool.New(),
		websearchtool.New(),
	}
	if features.ToolSearchEnabledOptimistic() {
		builtins = append(builtins, toolsearchtool.New())
	}
	reg := registry.New(builtins...)
	return &registryBashToolRunner{reg: reg}
}

type registryBashToolRunner struct {
	reg *registry.Registry
}

func (r *registryBashToolRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if r.reg.ByName(name) != nil {
		return r.reg.RunTool(ctx, name, inputJSON)
	}
	return BashExecToolRunner{}.RunTool(ctx, name, inputJSON)
}
