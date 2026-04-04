package compact

import (
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
	"github.com/2456868764/rabbit-code/internal/tools/notebookedittool"
	"github.com/2456868764/rabbit-code/internal/tools/webfetchtool"
	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
	"github.com/2456868764/rabbit-code/internal/utils/shell"
)

// ContextManagementConfig mirrors apiMicrocompact.ts ContextManagementConfig (JSON for API context_management).
type ContextManagementConfig struct {
	Edits []json.RawMessage `json:"edits"`
}

// APIContextManagementOptions mirrors getAPIContextManagement(options?).
type APIContextManagementOptions struct {
	HasThinking            bool
	IsRedactThinkingActive bool
	ClearAllThinking       bool
}

func toolsClearableResults() []string {
	out := append([]string{}, shell.ShellToolNames()...)
	out = append(out,
		globtool.GlobToolName,
		greptool.GrepToolName,
		filereadtool.FileReadToolName,
		webfetchtool.WebFetchToolName,
		websearchtool.WebSearchToolName,
	)
	return out
}

func toolsClearableUses() []string {
	return []string{
		fileedittool.FileEditToolName,
		filewritetool.FileWriteToolName,
		notebookedittool.NotebookEditToolName,
	}
}

// GetAPIContextManagement mirrors apiMicrocompact.ts getAPIContextManagement.
// Returns nil when there are no edits (TS returns undefined).
func GetAPIContextManagement(opts APIContextManagementOptions) *ContextManagementConfig {
	var edits []json.RawMessage

	if opts.HasThinking && !opts.IsRedactThinkingActive {
		var keep interface{}
		if opts.ClearAllThinking {
			keep = map[string]interface{}{
				"type":  "thinking_turns",
				"value": 1,
			}
		} else {
			keep = "all"
		}
		raw, err := json.Marshal(map[string]interface{}{
			"type": "clear_thinking_20251015",
			"keep": keep,
		})
		if err == nil {
			edits = append(edits, raw)
		}
	}

	if !features.AntUserType() {
		return finalizeContextManagement(edits)
	}

	triggerThreshold := features.APIMaxInputTokens()
	keepTarget := features.APITargetInputTokens()

	if features.UseAPIClearToolResults() {
		clearAtLeast := triggerThreshold - keepTarget
		if clearAtLeast < 0 {
			clearAtLeast = 0
		}
		raw, err := json.Marshal(map[string]interface{}{
			"type": "clear_tool_uses_20250919",
			"trigger": map[string]interface{}{
				"type":  "input_tokens",
				"value": triggerThreshold,
			},
			"clear_at_least": map[string]interface{}{
				"type":  "input_tokens",
				"value": clearAtLeast,
			},
			"clear_tool_inputs": toolsClearableResults(),
		})
		if err == nil {
			edits = append(edits, raw)
		}
	}

	if features.UseAPIClearToolUses() {
		clearAtLeast := triggerThreshold - keepTarget
		if clearAtLeast < 0 {
			clearAtLeast = 0
		}
		raw, err := json.Marshal(map[string]interface{}{
			"type": "clear_tool_uses_20250919",
			"trigger": map[string]interface{}{
				"type":  "input_tokens",
				"value": triggerThreshold,
			},
			"clear_at_least": map[string]interface{}{
				"type":  "input_tokens",
				"value": clearAtLeast,
			},
			"exclude_tools": toolsClearableUses(),
		})
		if err == nil {
			edits = append(edits, raw)
		}
	}

	return finalizeContextManagement(edits)
}

func finalizeContextManagement(edits []json.RawMessage) *ContextManagementConfig {
	if len(edits) == 0 {
		return nil
	}
	return &ContextManagementConfig{Edits: edits}
}
