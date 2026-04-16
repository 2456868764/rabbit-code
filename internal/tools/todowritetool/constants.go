// Package todowritetool mirrors restored-src/src/tools/TodoWriteTool (TodoWriteTool.ts).
package todowritetool

// TodoWriteToolName is TODO_WRITE_TOOL_NAME upstream.
const TodoWriteToolName = "TodoWrite"

// VerificationAgentType is VERIFICATION_AGENT_TYPE (AgentTool/constants.ts) for the closing-task nudge text.
const VerificationAgentType = "verification"

// Tool metadata constants from TodoWriteTool.ts buildTool definition.
const (
	// SearchHint mirrors searchHint in TodoWriteTool.ts.
	SearchHint = "manage the session task checklist"
	// MaxResultSizeChars mirrors maxResultSizeChars in TodoWriteTool.ts.
	MaxResultSizeChars = 100_000
	// ShouldDefer mirrors shouldDefer: true in TodoWriteTool.ts.
	ShouldDefer = true
	// Strict mirrors strict: true in TodoWriteTool.ts (schema strictness).
	Strict = true
)
