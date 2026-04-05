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

// NewDefaultToolRunner returns NewDefaultToolRunnerForModel("") (main loop model from ANTHROPIC_MODEL or default haiku).
// WebSearch is registered only when features.WebSearchToolEnabled(resolvedModel) matches WebSearchTool.isEnabled upstream.
func NewDefaultToolRunner() ToolRunner {
	return NewDefaultToolRunnerForModel("")
}

// NewDefaultToolRunnerForModel builds the default registry; mainLoopModel seeds WebSearch gating (Vertex 4.x, Bedrock off, etc.).
// and ToolSearch when features.ToolSearchEnabledOptimistic() matches upstream (utils/toolSearch.ts),
// plus BashExecToolRunner for tool name "bash" when not handled by the registry.
func NewDefaultToolRunnerForModel(mainLoopModel string) ToolRunner {
	builtins := []tools.Tool{
		filereadtool.New(),
		filewritetool.New(),
		fileedittool.New(),
		globtool.New(),
		greptool.New(),
		notebookedittool.New(),
		todowritetool.New(),
		webfetchtool.New(),
	}
	if features.WebSearchToolEnabled(ResolveMainLoopModel(mainLoopModel)) {
		builtins = append(builtins, websearchtool.New())
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
