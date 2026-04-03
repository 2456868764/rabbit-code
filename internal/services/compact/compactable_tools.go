package compact

// CompactableTool names mirror microCompact.ts COMPACTABLE_TOOLS (src/services/compact/microCompact.ts).
// Headless parity: used to classify which tool results may be micro-compacted upstream.
var compactableTools = map[string]struct{}{
	"Read":       {},
	"Bash":       {},
	"PowerShell": {},
	"Grep":       {},
	"Glob":       {},
	"WebSearch":  {},
	"WebFetch":   {},
	"Edit":       {},
	"Write":      {},
}

// IsCompactableToolName reports whether name is in the upstream COMPACTABLE_TOOLS set.
func IsCompactableToolName(name string) bool {
	_, ok := compactableTools[name]
	return ok
}
