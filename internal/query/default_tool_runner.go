package query

import (
	"context"

	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
	"github.com/2456868764/rabbit-code/internal/tools/registry"
)

// NewDefaultToolRunner returns a ToolRunner with Phase-6 builtins (Read, Write, Edit, Glob, Grep)
// plus BashExecToolRunner for tool name "bash" when not handled by the registry.
func NewDefaultToolRunner() ToolRunner {
	reg := registry.New(
		filereadtool.New(),
		filewritetool.New(),
		fileedittool.New(),
		globtool.New(),
		greptool.New(),
	)
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
