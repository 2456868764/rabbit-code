package thinking

import (
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// InterleavedAPIContextManagementOpts returns compact.APIContextManagementOptions for Messages API
// context_management (apiMicrocompact.ts / claude.ts interleaved thinking + redact + clear-all latch).
// Provider must stay aligned with anthropic.Provider numeric values.
func InterleavedAPIContextManagementOpts(model string, p Provider) compact.APIContextManagementOptions {
	return compact.APIContextManagementOptions{
		HasThinking:            ModelSupportsThinking(model, p) && ShouldEnableThinkingByDefault(),
		IsRedactThinkingActive: features.RedactThinkingEnabled(),
		ClearAllThinking:       features.ThinkingClearAllLatched(),
	}
}
