package anthropic

import (
	"bytes"

	"github.com/2456868764/rabbit-code/internal/features"
)

// IsPromptCacheBreakStreamJSON reports whether an SSE `data:` JSON line indicates a prompt cache
// break, when PROMPT_CACHE_BREAK_DETECTION is enabled. Heuristics extend as parity with
// promptCacheBreakDetection.ts improves.
func IsPromptCacheBreakStreamJSON(jsonLine []byte) bool {
	if !features.PromptCacheBreakDetection() {
		return false
	}
	b := bytes.ToLower(jsonLine)
	if bytes.Contains(b, []byte("cache_break")) {
		return true
	}
	if bytes.Contains(b, []byte("prompt_cache")) && bytes.Contains(b, []byte("break")) {
		return true
	}
	if bytes.Contains(b, []byte("prompt cache break")) {
		return true
	}
	if bytes.Contains(b, []byte("prompt_cache_invalid")) {
		return true
	}
	if bytes.Contains(b, []byte("invalidated_prompt")) {
		return true
	}
	if bytes.Contains(b, []byte("prompt_cache")) && bytes.Contains(b, []byte("expired")) {
		return true
	}
	if bytes.Contains(b, []byte("prompt_cache_miss")) {
		return true
	}
	if bytes.Contains(b, []byte("invalid_cache")) {
		return true
	}
	if bytes.Contains(b, []byte("cached_prompt")) && bytes.Contains(b, []byte("invalid")) {
		return true
	}
	if bytes.Contains(b, []byte("cache_key")) && bytes.Contains(b, []byte("invalid")) {
		return true
	}
	// Additional API / proxy phrasing (H1 parity with promptCacheBreakDetection.ts extensions).
	if bytes.Contains(b, []byte("prompt_cache_key")) && (bytes.Contains(b, []byte("invalid")) || bytes.Contains(b, []byte("unknown")) || bytes.Contains(b, []byte("missing"))) {
		return true
	}
	if bytes.Contains(b, []byte("cached")) && bytes.Contains(b, []byte("block")) && bytes.Contains(b, []byte("invalid")) {
		return true
	}
	if bytes.Contains(b, []byte("ephemeral")) && bytes.Contains(b, []byte("cache")) && bytes.Contains(b, []byte("stale")) {
		return true
	}
	return false
}
