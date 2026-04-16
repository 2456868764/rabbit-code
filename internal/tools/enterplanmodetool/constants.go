// Package enterplanmodetool implements the EnterPlanMode tool (P6.4).
package enterplanmodetool

// EnterPlanModeToolName is ENTER_PLAN_MODE_TOOL_NAME upstream.
const EnterPlanModeToolName = "EnterPlanMode"

// Tool metadata constants from EnterPlanModeTool.ts buildTool definition.
const (
	// SearchHint mirrors searchHint in EnterPlanModeTool.ts.
	SearchHint = "switch to plan mode to design an approach before coding"
	// MaxResultSizeChars mirrors maxResultSizeChars in EnterPlanModeTool.ts.
	MaxResultSizeChars = 100_000
	// ShouldDefer mirrors shouldDefer: true in EnterPlanModeTool.ts.
	ShouldDefer = true
	// IsConcurrencySafe mirrors isConcurrencySafe() → true in EnterPlanModeTool.ts.
	IsConcurrencySafe = true
	// IsReadOnly mirrors isReadOnly() → true in EnterPlanModeTool.ts.
	IsReadOnly = true
	// ToolDescription mirrors description() in EnterPlanModeTool.ts.
	ToolDescription = "Requests permission to enter plan mode for complex tasks requiring exploration and design"
)
