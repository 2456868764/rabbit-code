package app

import (
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/query/engine"
)

// ApplyEngineCompactIntegration runs compact wiring before engine.New, alongside ApplyEngineMemdirFromRuntime.
// rt and cfg are reserved for future hooks (logging, defaults); EnsureForkPartialFromForkCompactSummary(aa) runs whenever aa is non-nil.
func ApplyEngineCompactIntegration(rt *Runtime, cfg *engine.Config, aa *query.AnthropicAssistant) {
	_ = rt
	_ = cfg
	query.EnsureForkPartialFromForkCompactSummary(aa)
}
