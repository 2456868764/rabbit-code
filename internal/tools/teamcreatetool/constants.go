// Package teamcreatetool implements the TeamCreate tool (P6.5).
package teamcreatetool

// TeamCreateToolName is TEAM_CREATE_TOOL_NAME upstream.
const TeamCreateToolName = "TeamCreate"

// Tool metadata constants from TeamCreateTool.ts buildTool definition.
const (
	// SearchHint mirrors searchHint in TeamCreateTool.ts.
	SearchHint = "create a multi-agent swarm team"
	// MaxResultSizeChars mirrors maxResultSizeChars in TeamCreateTool.ts.
	MaxResultSizeChars = 100_000
	// ShouldDefer mirrors shouldDefer: true in TeamCreateTool.ts.
	ShouldDefer = true
)
