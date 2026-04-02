package cost

// Merge adds b into a (for accumulating across deltas / server_tool_use).
func Merge(a, b Usage) Usage {
	out := a
	out.InputTokens += b.InputTokens
	out.CacheCreationInputTokens += b.CacheCreationInputTokens
	out.CacheReadInputTokens += b.CacheReadInputTokens
	out.OutputTokens += b.OutputTokens
	out.WebSearchRequests += b.WebSearchRequests
	out.WebFetchRequests += b.WebFetchRequests
	out.Ephemeral1hInputTokens += b.Ephemeral1hInputTokens
	out.Ephemeral5mInputTokens += b.Ephemeral5mInputTokens
	if b.ServiceTier != "" {
		out.ServiceTier = b.ServiceTier
	}
	if b.InferenceGeo != "" {
		out.InferenceGeo = b.InferenceGeo
	}
	if b.Speed != "" {
		out.Speed = b.Speed
	}
	return out
}
