package compact

import (
	"context"
	"encoding/json"
)

type executorSuggestKey struct{}

// ExecutorSuggestMeta carries compact suggest flags from engine into CompactExecutor.
// After ExecutorPhaseAfterSchedule the RunPhase argument is always RunExecuting, so hooks use this meta for auto vs reactive (compact.ts trigger: 'auto' | 'manual').
type ExecutorSuggestMeta struct {
	AutoCompact     bool
	ReactiveCompact bool
}

// ContextWithExecutorSuggestMeta attaches meta for querydeps / app CompactExecutor closures.
func ContextWithExecutorSuggestMeta(ctx context.Context, m ExecutorSuggestMeta) context.Context {
	return context.WithValue(ctx, executorSuggestKey{}, m)
}

// ExecutorSuggestMetaFromContext returns meta attached by the engine; ok is false if absent.
func ExecutorSuggestMetaFromContext(ctx context.Context) (ExecutorSuggestMeta, bool) {
	v, ok := ctx.Value(executorSuggestKey{}).(ExecutorSuggestMeta)
	return v, ok
}

// ToolSearchToolName mirrors ToolSearchTool/constants.ts TOOL_SEARCH_TOOL_NAME.
const ToolSearchToolName = "ToolSearch"

// DefaultCompactStreamingToolsJSON mirrors compact.ts streamCompactSummary tools slice (minimal Read ± ToolSearch for deferred-tool parity).
func DefaultCompactStreamingToolsJSON(includeToolSearch bool) (json.RawMessage, error) {
	read := map[string]interface{}{
		"name":        "Read",
		"description": "Read a file from the local filesystem.",
		"input_schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]string{"type": "string", "description": "Absolute or workspace-relative file path"},
			},
			"required": []string{"file_path"},
		},
	}
	list := []interface{}{read}
	if includeToolSearch {
		list = append(list, map[string]interface{}{
			"name":        ToolSearchToolName,
			"description": "Search for a tool definition matching the query.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]string{"type": "string", "description": "Search query"},
				},
				"required": []string{"query"},
			},
		})
	}
	return json.Marshal(list)
}
