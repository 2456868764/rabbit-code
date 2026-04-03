package query

import "strings"

// UserTextHintFlags gates optional user-message hints (P5.F.3–F.5, headless).
type UserTextHintFlags struct {
	ContextCollapse bool
	Ultrathink      bool
	Ultraplan       bool
	SessionRestore  bool
}

const (
	hintContextCollapseSuffix = "\n\n[CONTEXT_COLLAPSE: prefer collapsing stale context; avoid repeating large verbatim dumps.]\n"
	hintUltrathinkPrefix      = "[ULTRATHINK: reason step-by-step before answering.]\n\n"
	hintUltraplanSuffix       = "\n\n[ULTRAPLAN: outline a short plan before executing tool calls.]"
	hintSessionRestoreSuffix  = "\n\n[SESSION_RESTORE: prefer restoring durable session context over re-deriving from scratch.]"
)

// ApplyUserTextHints mutates the resolved user payload before InitialUserMessagesJSON (engine Submit path).
func ApplyUserTextHints(text string, f UserTextHintFlags) string {
	if text == "" {
		return text
	}
	out := text
	if f.Ultrathink {
		out = hintUltrathinkPrefix + out
	}
	if f.ContextCollapse {
		out = out + hintContextCollapseSuffix
	}
	if f.Ultraplan {
		out = out + hintUltraplanSuffix
	}
	if f.SessionRestore {
		out = out + hintSessionRestoreSuffix
	}
	return out
}

// FormatHeadlessModeTags lists active headless input modes for TUI/telemetry (comma-separated, stable order).
func FormatHeadlessModeTags(f UserTextHintFlags) string {
	var parts []string
	if f.ContextCollapse {
		parts = append(parts, "context_collapse")
	}
	if f.Ultrathink {
		parts = append(parts, "ultrathink")
	}
	if f.Ultraplan {
		parts = append(parts, "ultraplan")
	}
	if f.SessionRestore {
		parts = append(parts, "session_restore")
	}
	return strings.Join(parts, ",")
}
