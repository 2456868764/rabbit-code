// Package exitplanmodetool implements the ExitPlanMode (V2) tool (P6.4).
package exitplanmodetool

// ExitPlanModeToolName is EXIT_PLAN_MODE_V2_TOOL_NAME upstream (both constants map to "ExitPlanMode").
const ExitPlanModeToolName = "ExitPlanMode"

// Tool metadata constants from ExitPlanModeV2Tool.ts buildTool definition.
const (
	// SearchHint mirrors searchHint in ExitPlanModeV2Tool.ts.
	SearchHint = "present plan for approval and start coding (plan mode only)"
	// MaxResultSizeChars mirrors maxResultSizeChars in ExitPlanModeV2Tool.ts.
	MaxResultSizeChars = 100_000
	// ShouldDefer mirrors shouldDefer: true in ExitPlanModeV2Tool.ts.
	ShouldDefer = true
	// ToolDescription mirrors description() in ExitPlanModeV2Tool.ts.
	ToolDescription = "Prompts the user to exit plan mode and start coding"
)
