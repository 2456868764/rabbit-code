package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

// BashStubToolRunner is a Phase 5 tool runner that accepts only "bash" and returns a fixed JSON result (P5.1.2 bridge until Phase 6).
type BashStubToolRunner struct{}

// RunTool implements ToolRunner.
func (BashStubToolRunner) RunTool(_ context.Context, name string, inputJSON []byte) ([]byte, error) {
	if name != "bash" {
		return nil, fmt.Errorf("query: bash stub: unknown tool %q", name)
	}
	_ = inputJSON
	return json.RawMessage(`{"ok":true,"stub":"bash"}`), nil
}

// BashExecToolRunner runs Bash tool calls through bashtool.Bash when RABBIT_CODE_BASH_EXEC is truthy; otherwise delegates to BashStubToolRunner.
// Accepts tool name "bash" or "Bash". Normalizes legacy {"cmd":...} to {"command":...}. No read-only gate here (PARITY_H9_BASH_PERMISSIONS.md §4).
type BashExecToolRunner struct{}

// RunTool implements ToolRunner.
func (BashExecToolRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if name != "bash" && name != bashtool.BashToolName {
		return nil, fmt.Errorf("query: bash exec: unknown tool %q", name)
	}
	if !features.BashExecEnabled() {
		return BashStubToolRunner{}.RunTool(ctx, "bash", inputJSON)
	}
	body := bashtool.NormalizeLegacyBashInput(inputJSON)
	return bashtool.New().Run(ctx, body)
}
