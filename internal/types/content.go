package types

import "encoding/json"

// ContentPiece is a discriminated JSON object in message content (type field).
// Optional fields use omitempty; only relevant fields are set per Type.
//
// API-oriented types: "text", "tool_use", "tool_result", "document" (source.*).
// Internal / feature extensions: "boundary", "tombstone", "history_snip",
// "connector_text", "compaction_reminder", "file_ref",
// "kairos_queue", "kairos_channel", "kairos_brief", "uds_inbox", "progress".
type ContentPiece struct {
	Type string `json:"type"`

	// text, connector_text (same Text field; connector_text normalizes to text for API)
	Text string `json:"text,omitempty"`

	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// tool_result
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   *bool           `json:"is_error,omitempty"`

	// document (Messages API; e.g. from file_ref URL mapping in messages.NormalizeForAPI)
	Source json.RawMessage `json:"source,omitempty"`

	// boundary (compact / session markers, P3.2.3)
	Kind string `json:"kind,omitempty"`

	// tombstone
	TombstoneID string `json:"tombstone_id,omitempty"`

	// history_snip (P3.F.1)
	SnipID   string `json:"snip_id,omitempty"`
	SnipEdge string `json:"snip_edge,omitempty"` // "start" | "end"

	// compaction_reminder (P3.F.3)
	ReminderID string `json:"reminder_id,omitempty"`

	// file_ref large attachment (P3.3.2)
	Ref       string `json:"ref,omitempty"`
	Sha256    string `json:"sha256,omitempty"`
	MediaType string `json:"media_type,omitempty"`

	// kairos (P3.F.4–F.6)
	QueueID   string `json:"queue_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	PlanID    string `json:"plan_id,omitempty"`
	BriefID   string `json:"brief_id,omitempty"`

	// uds_inbox (P3.F.7)
	InboxAddress string `json:"inbox_address,omitempty"`

	// progress (P3.1.2)
	Label   string `json:"label,omitempty"`
	Percent *int   `json:"percent,omitempty"`
}

const (
	BlockTypeText                = "text"
	BlockTypeToolUse             = "tool_use"
	BlockTypeToolResult          = "tool_result"
	BlockTypeBoundary            = "boundary"
	BlockTypeTombstone           = "tombstone"
	BlockTypeHistorySnip         = "history_snip"
	BlockTypeConnectorText       = "connector_text"
	BlockTypeCompactionReminder  = "compaction_reminder"
	BlockTypeFileRef             = "file_ref"
	BlockTypeDocument            = "document"
	BlockTypeKairosQueue         = "kairos_queue"
	BlockTypeKairosChannel       = "kairos_channel"
	BlockTypeKairosBrief         = "kairos_brief"
	BlockTypeUDSInbox            = "uds_inbox"
	BlockTypeProgress            = "progress"
)
