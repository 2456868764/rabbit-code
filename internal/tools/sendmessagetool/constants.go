// Package sendmessagetool implements the SendMessage tool (P6.5).
package sendmessagetool

// SendMessageToolName is SEND_MESSAGE_TOOL_NAME upstream.
const SendMessageToolName = "SendMessage"

// TeamLeadName is TEAM_LEAD_NAME upstream (swarm/constants.ts).
const TeamLeadName = "team-lead"

// Tool metadata from SendMessageTool.ts buildTool definition.
const (
	// Description mirrors DESCRIPTION in SendMessageTool/prompt.ts.
	Description = "Send a message to another agent"
	// MaxResultSizeChars — not explicitly set in TS; defaults to 100_000.
	MaxResultSizeChars = 100_000
)
