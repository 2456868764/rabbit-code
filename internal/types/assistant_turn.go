package types

import "encoding/json"

// ToolUseCall is one tool_use block in an assistant message (Messages API shape).
// Shared by internal/query and internal/services/api (AnthropicAssistant) without import cycles.
type ToolUseCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// TurnResult is one assistant model response (text + optional tool uses).
type TurnResult struct {
	Text       string
	ToolUses   []ToolUseCall
	StopReason string
}
