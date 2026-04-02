package cost

// FromUsageDelta maps Anthropic Messages stream usage fields into Usage (services/api/usage.ts).
func FromUsageDelta(input, cacheCreate, cacheRead, output int64) Usage {
	return Usage{
		InputTokens:              input,
		CacheCreationInputTokens: cacheCreate,
		CacheReadInputTokens:     cacheRead,
		OutputTokens:             output,
	}
}
