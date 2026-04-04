package app

import (
	"github.com/2456868764/rabbit-code/internal/query/engine"
	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

// ApplyEngineCompactIntegration runs compact wiring before engine.New, alongside ApplyEngineMemdirFromRuntime.
// rt and cfg are reserved for future hooks (logging, defaults); EnsureForkPartialFromForkCompactSummary(aa) runs whenever aa is non-nil.
func ApplyEngineCompactIntegration(rt *Runtime, cfg *engine.Config, aa *querydeps.AnthropicAssistant) {
	_ = rt
	_ = cfg
	querydeps.EnsureForkPartialFromForkCompactSummary(aa)
}
