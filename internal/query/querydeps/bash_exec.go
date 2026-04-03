package querydeps

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

// BashExecToolRunner runs bash tool calls via sh -c when RABBIT_CODE_BASH_EXEC is truthy; otherwise delegates to BashStubToolRunner.
type BashExecToolRunner struct{}

// RunTool implements ToolRunner.
func (BashExecToolRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if name != "bash" {
		return nil, fmt.Errorf("querydeps: bash exec: unknown tool %q", name)
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
