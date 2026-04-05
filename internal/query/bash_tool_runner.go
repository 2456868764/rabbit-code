package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
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

// BashExecToolRunner runs bash tool calls via sh -c when RABBIT_CODE_BASH_EXEC is truthy; otherwise delegates to BashStubToolRunner.
// Rejects NUL in the command string (H9 hygiene). No read-only gate here; extract path uses memdir.IsExtractReadOnlyBash (PARITY_H9_BASH_PERMISSIONS.md §4).
type BashExecToolRunner struct{}

// RunTool implements ToolRunner.
func (BashExecToolRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if name != "bash" {
		return nil, fmt.Errorf("query: bash exec: unknown tool %q", name)
	}
	if !features.BashExecEnabled() {
		return BashStubToolRunner{}.RunTool(ctx, name, inputJSON)
	}
	var in struct {
		Command string `json:"command"`
		Cmd     string `json:"cmd"`
	}
	_ = json.Unmarshal(inputJSON, &in)
	shell := strings.TrimSpace(in.Command)
	if shell == "" {
		shell = strings.TrimSpace(in.Cmd)
	}
	if shell == "" {
		return json.Marshal(map[string]any{"ok": true, "stdout": "", "stderr": "", "exit": 0})
	}
	if strings.ContainsRune(shell, 0) {
		return nil, fmt.Errorf("query: bash exec: null byte in command")
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "sh", "-c", shell)
	out, err := cmd.CombinedOutput()
	exit := 0
	ok := err == nil
	if err != nil {
		exit = 1
	}
	return json.Marshal(map[string]any{"ok": ok, "stdout": string(out), "stderr": "", "exit": exit})
}
