package memdir

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

// AutoMemToolRunner wraps a ToolRunner and enforces createAutoMemCanUseTool rules (extractMemories.ts).
type AutoMemToolRunner struct {
	Inner     querydeps.ToolRunner
	MemoryDir string
}

// RunTool implements querydeps.ToolRunner.
func (w *AutoMemToolRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if w.Inner == nil {
		return nil, querydeps.ErrNoToolRunner
	}
	memRoot := strings.TrimSpace(w.MemoryDir)
	n := strings.TrimSpace(name)

	switch {
	case strings.EqualFold(n, "Read"), strings.EqualFold(n, "Grep"), strings.EqualFold(n, "Glob"):
		return w.Inner.RunTool(ctx, name, inputJSON)
	case strings.EqualFold(n, "REPL"):
		// extractMemories.ts: allow REPL so cache-safe tool lists match forkedAgent; inner tools stay gated.
		return w.Inner.RunTool(ctx, name, inputJSON)
	case strings.EqualFold(n, "bash"), strings.EqualFold(n, "Bash"):
		if IsExtractReadOnlyBash(inputJSON) {
			return w.Inner.RunTool(ctx, name, inputJSON)
		}
		return nil, fmt.Errorf("memdir: only read-only shell commands are permitted in this context")
	case strings.EqualFold(n, "Write"), strings.EqualFold(n, "Edit"):
		fp, ok := jsonFilePath(inputJSON)
		if ok && memRoot != "" && IsAutoMemPath(fp, memRoot) {
			return w.Inner.RunTool(ctx, name, inputJSON)
		}
		return nil, fmt.Errorf("memdir: Write/Edit only allowed under auto-memory directory")
	default:
		return nil, fmt.Errorf("memdir: tool %q denied in extract context", name)
	}
}

func jsonFilePath(inputJSON []byte) (string, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(inputJSON, &m); err != nil {
		return "", false
	}
	v, ok := m["file_path"]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}
