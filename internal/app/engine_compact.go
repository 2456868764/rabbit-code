package app

import (
	"strings"

	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/query/engine"
	"github.com/2456868764/rabbit-code/internal/utils/thinking"
)

// ApplyEngineCompactIntegration runs compact wiring before engine.New, alongside ApplyEngineMemdirFromRuntime.
// rt is reserved for future hooks. When aa has a Client and no APIContextManagementOpts, defaults mirror
// apiMicrocompact.ts interleaved thinking + env redact/clear-all (thinking.InterleavedAPIContextManagementOpts).
func ApplyEngineCompactIntegration(rt *Runtime, cfg *engine.Config, aa *anthropic.AnthropicAssistant) {
	_ = rt
	anthropic.EnsureForkPartialFromForkCompactSummary(aa)
	if aa == nil || aa.Client == nil || aa.APIContextManagementOpts != nil {
		return
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = strings.TrimSpace(aa.DefaultModel)
	}
	opts := thinking.InterleavedAPIContextManagementOpts(model, thinking.Provider(aa.Client.Provider))
	aa.APIContextManagementOpts = &opts
}
