package query

import "strings"

// Phase5UserTextFlags gates optional user-message hints (P5.F.3–F.5).
type Phase5UserTextFlags struct {
	ContextCollapse bool
	Ultrathink      bool
	Ultraplan       bool
	SessionRestore  bool
}

const (
	phase5ContextCollapseSuffix = "\n\n[CONTEXT_COLLAPSE: prefer collapsing stale context; avoid repeating large verbatim dumps.]\n"
	phase5UltrathinkPrefix      = "[ULTRATHINK: reason step-by-step before answering.]\n\n"
	phase5UltraplanSuffix       = "\n\n[ULTRAPLAN: outline a short plan before executing tool calls.]"
	phase5SessionRestoreSuffix  = "\n\n[SESSION_RESTORE: prefer restoring durable session context over re-deriving from scratch.]"
)

// ApplyPhase5UserTextHints mutates the resolved user payload before InitialUserMessagesJSON (engine Submit path).
func ApplyPhase5UserTextHints(text string, f Phase5UserTextFlags) string {
	if text == "" {
		return text
	}
	out := text
	if f.Ultrathink {
		out = phase5UltrathinkPrefix + out
	}
	if f.ContextCollapse {
		out = out + phase5ContextCollapseSuffix
	}
	if f.Ultraplan {
		out = out + phase5UltraplanSuffix
	}
	if f.SessionRestore {
		out = out + phase5SessionRestoreSuffix
	}
	return out
}

// FormatPhase5HeadlessModeTags lists active Phase 5 input modes for TUI/telemetry (comma-separated, stable order).
func FormatPhase5HeadlessModeTags(f Phase5UserTextFlags) string {
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
