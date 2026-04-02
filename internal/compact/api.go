// Package compact holds API-adjacent compact hooks for Phase 4 (CACHED_MICROCOMPACT, PROMPT_CACHE_BREAK_DETECTION).
// Phase 5 adds RunPhase scheduling skeleton (phase.go); full auto/reactive machine still expands here.
package compact

import (
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// MicroCompactRequested mirrors CACHED_MICROCOMPACT gating at HTTP boundary (services/compact/apiMicrocompact.ts).
func MicroCompactRequested() bool {
	return strings.TrimSpace(os.Getenv("RABBIT_CODE_CACHED_MICROCOMPACT")) == "1"
}

// PromptCacheBreakActive mirrors PROMPT_CACHE_BREAK_DETECTION (services/api/promptCacheBreakDetection.ts).
func PromptCacheBreakActive() bool {
	return features.PromptCacheBreakDetection()
}
