package cost

import "github.com/2456868764/rabbit-code/internal/bootstrap"

// ApplyUsageToBootstrap writes token fields into bootstrap.State (costHook / claude.ts alignment).
func ApplyUsageToBootstrap(st *bootstrap.State, u Usage) {
	if st == nil {
		return
	}
	st.RecordTokenUsage(bootstrap.TokenUsage{
		InputTokens:              u.InputTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:     u.CacheReadInputTokens,
		OutputTokens:             u.OutputTokens,
	})
}
