package query

import "github.com/2456868764/rabbit-code/internal/features"

// Query source strings mirror query.ts QuerySource where they gate shouldAutoCompact (autoCompact.ts).
const (
	QuerySourceSessionMemory   = "session_memory"
	QuerySourceCompact         = "compact"
	QuerySourceMarbleOrigami   = "marble_origami"
	QuerySourceExtractMemories = "extract_memories"
)

// ProactiveAutoCompactAllowedForQuerySource is false for forked agents that would deadlock or corrupt
// shared state when proactive autocompact runs (session_memory, compact), and for marble_origami when
// CONTEXT_COLLAPSE is enabled (autoCompact.ts).
func ProactiveAutoCompactAllowedForQuerySource(source string) bool {
	switch source {
	case QuerySourceSessionMemory, QuerySourceCompact, QuerySourceExtractMemories:
		return false
	}
	if source == QuerySourceMarbleOrigami && features.ContextCollapseEnabled() {
		return false
	}
	return true
}
