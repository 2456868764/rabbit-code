package anthropic

import (
	"github.com/2456868764/rabbit-code/internal/bootstrap"
	"github.com/2456868764/rabbit-code/internal/cost"
)

// SetOnStreamUsageBootstrap sets OnStreamUsage to apply final stream UsageDelta via cost.ApplyUsageToBootstrap (P4.4.1).
// Nil c or st is a no-op. Replaces any existing OnStreamUsage.
func (c *Client) SetOnStreamUsageBootstrap(st *bootstrap.State) {
	if c == nil || st == nil {
		return
	}
	c.OnStreamUsage = func(u UsageDelta) {
		cost.ApplyUsageToBootstrap(st, cost.FromUsageDelta(
			u.InputTokens,
			u.CacheCreationInputTokens,
			u.CacheReadInputTokens,
			u.OutputTokens,
		))
	}
}
