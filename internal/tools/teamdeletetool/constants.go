// Package teamdeletetool implements the TeamDelete tool (P6.5).
package teamdeletetool

// TeamDeleteToolName is TEAM_DELETE_TOOL_NAME upstream.
const TeamDeleteToolName = "TeamDelete"

// Tool metadata constants from TeamDeleteTool.ts buildTool definition.
const (
	// SearchHint mirrors searchHint in TeamDeleteTool.ts.
	SearchHint = "disband a swarm team and clean up"
	// MaxResultSizeChars mirrors maxResultSizeChars in TeamDeleteTool.ts.
	MaxResultSizeChars = 100_000
	// ShouldDefer mirrors shouldDefer: true in TeamDeleteTool.ts.
	ShouldDefer = true
	// ToolDescription mirrors description() in TeamDeleteTool.ts.
	ToolDescription = "Clean up team and task directories when the swarm is complete"
)
