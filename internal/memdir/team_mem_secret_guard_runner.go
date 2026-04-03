package memdir

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

// TeamMemSecretGuardRunner blocks Write/Edit when content matches teamMemSecretScan (teamMemSecretGuard.ts).
type TeamMemSecretGuardRunner struct {
	Inner      querydeps.ToolRunner
	AutoMemDir string
	Enabled    bool
}

// RunTool implements querydeps.ToolRunner.
func (w *TeamMemSecretGuardRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if w.Inner == nil {
		return nil, querydeps.ErrNoToolRunner
	}
	if w.Enabled && w.AutoMemDir != "" {
		if fp, content, ok := toolWritePathAndContent(name, inputJSON); ok && content != "" {
			if abs, err := filepath.Abs(fp); err == nil && IsTeamMemPathUnderAutoMem(abs, w.AutoMemDir+string(filepath.Separator)) {
				if err := ValidateTeamMemWritePath(abs, w.AutoMemDir); err != nil {
					return nil, err
				}
				if matches := ScanTeamMemorySecrets(content); len(matches) > 0 {
					var labels []string
					for _, m := range matches {
						labels = append(labels, m.Label)
					}
					return nil, fmt.Errorf(
						"memdir: content contains potential secrets (%s) and cannot be written to team memory. "+
							"Team memory is shared with all repository collaborators. Remove the sensitive content and try again",
						strings.Join(labels, ", "),
					)
				}
			}
		}
	}
	return w.Inner.RunTool(ctx, name, inputJSON)
}

func toolWritePathAndContent(toolName string, inputJSON []byte) (filePath string, content string, ok bool) {
	n := strings.ToLower(strings.TrimSpace(toolName))
	if n != "write" && n != "edit" {
		return "", "", false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(inputJSON, &m); err != nil {
		return "", "", false
	}
	fp := jsonStringField(m["file_path"])
	if fp == "" {
		return "", "", false
	}
	switch n {
	case "write":
		return fp, jsonStringField(m["content"]), true
	case "edit":
		oldS := jsonStringField(m["old_string"])
		newS := jsonStringField(m["new_string"])
		return fp, oldS + "\n" + newS, true
	default:
		return "", "", false
	}
}

func jsonStringField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}
