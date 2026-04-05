package query

import (
	"os"
	"strings"
)

// ResolveMainLoopModel returns explicit trimmed model when non-empty, else ANTHROPIC_MODEL, else engine default.
func ResolveMainLoopModel(explicit string) string {
	if s := strings.TrimSpace(explicit); s != "" {
		return s
	}
	if s := strings.TrimSpace(os.Getenv("ANTHROPIC_MODEL")); s != "" {
		return s
	}
	return "claude-3-5-haiku-20241022"
}
