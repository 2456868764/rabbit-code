// HISTORY_SNIP / snipCompact parity (src/services/compact/snipCompact + messages.ts mergeUserMessages / appendMessageTag).
package messages

import (
	"os"
	"strings"
	"testing"
)

// IsSnipRuntimeEnabled mirrors TS isSnipRuntimeEnabled when snipCompact is not bundled:
// requires RABBIT_HISTORY_SNIP=1; disable with RABBIT_SNIP_RUNTIME_ENABLED=0.
func IsSnipRuntimeEnabled() bool {
	if os.Getenv("RABBIT_HISTORY_SNIP") != "1" {
		return false
	}
	if os.Getenv("RABBIT_SNIP_RUNTIME_ENABLED") == "0" {
		return false
	}
	return true
}

// SnipNudgeText mirrors TS SNIP_NUDGE_TEXT for context_efficiency attachment (override via RABBIT_SNIP_NUDGE_TEXT).
func SnipNudgeText() string {
	if s := strings.TrimSpace(os.Getenv("RABBIT_SNIP_NUDGE_TEXT")); s != "" {
		return s
	}
	return "Your conversation is long. When context is tight, you can remove older turns with the snip tool so the session stays within limits—prefer snipping over losing task state."
}

func shouldAppendSnipMessageTags() bool {
	if os.Getenv("RABBIT_HISTORY_SNIP") != "1" {
		return false
	}
	if !IsSnipRuntimeEnabled() {
		return false
	}
	return !testing.Testing()
}
