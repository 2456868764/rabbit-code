package querydeps

import (
	"context"
	"encoding/json"
	"fmt"
)

// BashStubToolRunner is a Phase 5 tool runner that accepts only "bash" and returns a fixed JSON result (P5.1.2 bridge until Phase 6).
type BashStubToolRunner struct{}

// RunTool implements ToolRunner.
func (BashStubToolRunner) RunTool(_ context.Context, name string, inputJSON []byte) ([]byte, error) {
	if name != "bash" {
		return nil, fmt.Errorf("querydeps: bash stub: unknown tool %q", name)
	}
	_ = inputJSON
	return json.RawMessage(`{"ok":true,"stub":"bash"}`), nil
}
