package compact

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/features"
)

// AutoCompactIfNeededInput mirrors autoCompact.ts autoCompactIfNeeded parameters (headless; legacy compactConversation runs outside).
type AutoCompactIfNeededInput struct {
	Ctx context.Context

	TranscriptJSON  json.RawMessage
	Model           string
	MaxOutputTokens int
	// ContextWindowTokens 0 → features.ContextWindowTokensForModel(Model) + ApplyAutoCompactWindowCap.
	ContextWindowTokens int
	QuerySource         string
	AgentID             string
	SnipTokensFreed     int

	// TokenUsage is proactive transcript usage (engine: structured estimate or byte/4).
	TokenUsage int

	// TrackingConsecutiveFailures is autoCompact.ts tracking.consecutiveFailures before this attempt.
	TrackingConsecutiveFailures int

	// ForceAuto bypasses token threshold when true (CompactAdvisor analogue; still respects circuit + preflight).
	ForceAuto bool

	// SessionMemoryCompact optional; nil skips trySessionMemoryCompaction.
	SessionMemoryCompact func(ctx context.Context, agentID, model string, autoCompactThreshold int, transcriptJSON json.RawMessage) (replacement json.RawMessage, ok bool, err error)

	// AfterSessionMemorySuccess optional: notifyCompaction / markPostCompaction (autoCompact.ts lines 302–305).
	AfterSessionMemorySuccess func(querySource, agentID string)
}

// AutoCompactIfNeededResult is the outcome before legacy compactConversation (engine runs CompactExecutor when RunLegacyAutoCompact).
type AutoCompactIfNeededResult struct {
	Skipped bool
	// SkipReason: disable_compact | circuit | below_threshold
	SkipReason string

	// SessionMemoryApplied when replacement transcript should replace loop messages.
	SessionMemoryApplied bool
	NewTranscript        json.RawMessage

	// RunLegacyAutoCompact when gates passed and session memory did not replace (caller runs compactExecutor).
	RunLegacyAutoCompact bool
	AutoCompactThreshold int
}

// AutoCompactIfNeeded mirrors autoCompact.ts autoCompactIfNeeded through trySessionMemoryCompaction inclusive.
// Legacy compactConversation is invoked by the caller when RunLegacyAutoCompact is true (engine CompactExecutor).
func AutoCompactIfNeeded(in AutoCompactIfNeededInput) (AutoCompactIfNeededResult, error) {
	if in.Ctx == nil {
		in.Ctx = context.Background()
	}
	if features.DisableCompact() {
		return AutoCompactIfNeededResult{Skipped: true, SkipReason: "disable_compact"}, nil
	}
	if in.TrackingConsecutiveFailures >= MaxConsecutiveAutocompactFailures {
		return AutoCompactIfNeededResult{Skipped: true, SkipReason: "circuit"}, nil
	}

	cw := in.ContextWindowTokens
	if cw <= 0 {
		cw = features.ContextWindowTokensForModel(in.Model)
	}
	cw = features.ApplyAutoCompactWindowCap(cw)

	tok := in.TokenUsage - in.SnipTokensFreed
	if tok < 0 {
		tok = 0
	}
	should := in.ForceAuto || ProactiveAutocompactFromUsage(tok, in.Model, in.MaxOutputTokens, cw, in.QuerySource)
	if !should {
		return AutoCompactIfNeededResult{Skipped: true, SkipReason: "below_threshold"}, nil
	}

	th := AutoCompactThresholdForProactive(in.Model, in.MaxOutputTokens, cw)
	out := AutoCompactIfNeededResult{
		RunLegacyAutoCompact: true,
		AutoCompactThreshold: th,
	}

	if in.SessionMemoryCompact != nil && th > 0 {
		rep, ok, smErr := in.SessionMemoryCompact(in.Ctx, in.AgentID, in.Model, th, in.TranscriptJSON)
		if smErr == nil && ok && len(bytes.TrimSpace(rep)) > 0 {
			if in.AfterSessionMemorySuccess != nil {
				in.AfterSessionMemorySuccess(in.QuerySource, in.AgentID)
			}
			return AutoCompactIfNeededResult{
				SessionMemoryApplied: true,
				NewTranscript:        json.RawMessage(append([]byte(nil), rep...)),
				AutoCompactThreshold: th,
			}, nil
		}
	}

	return out, nil
}
