package engine

import (
	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

// InstallAnthropicStreamingCompact sets CompactExecutor to StreamingCompactExecutorWithConfig with ReturnNextTranscript
// and merges AttachPostCompactToStreamingConfig when e is non-nil. It is a no-op if aa is nil or CompactExecutor was
// already set via Config. Call after New and before Submit (typical order: New → Install → goroutine events).
func (e *Engine) InstallAnthropicStreamingCompact(aa *querydeps.AnthropicAssistant, customInstructions string) {
	if e == nil || aa == nil || e.compactExecutor != nil {
		return
	}
	cfg := querydeps.StreamingCompactExecutorConfig{
		CustomInstructions:   customInstructions,
		ReturnNextTranscript: true,
	}
	e.AttachPostCompactToStreamingConfig(&cfg)
	e.compactExecutor = querydeps.StreamingCompactExecutorWithConfig(aa, cfg)
}
