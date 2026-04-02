// Package cost mirrors cost-tracker.ts / services/api/usage.ts / emptyUsage.ts token shapes for Phase 4.
package cost

// Usage is a Go view of Anthropic message usage (beta), aligned with EMPTY_USAGE in emptyUsage.ts.
type Usage struct {
	InputTokens              int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
	OutputTokens             int64
	WebSearchRequests        int64
	WebFetchRequests         int64
	ServiceTier              string
	InferenceGeo             string
	Speed                    string
	Ephemeral1hInputTokens   int64
	Ephemeral5mInputTokens   int64
}
