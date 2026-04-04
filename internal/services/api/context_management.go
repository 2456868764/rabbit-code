package anthropic

import (
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// ModelSupportsContextManagement mirrors utils/betas.ts modelSupportsContextManagement (Claude 4+ / provider heuristics).
func ModelSupportsContextManagement(model string, p Provider) bool {
	if p == ProviderFoundry {
		return true
	}
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	switch p {
	case ProviderAnthropic:
		return !strings.Contains(m, "claude-3-")
	default:
		return strings.Contains(m, "claude-opus-4") ||
			strings.Contains(m, "claude-sonnet-4") ||
			strings.Contains(m, "claude-haiku-4")
	}
}

// ShouldAttachContextManagementBeta mirrors when TS pushes CONTEXT_MANAGEMENT_BETA_HEADER (betas.ts):
// model supports context management, or ant with USE_API_CONTEXT_MANAGEMENT.
func ShouldAttachContextManagementBeta(model string, p Provider) bool {
	if ModelSupportsContextManagement(model, p) {
		return true
	}
	return features.UseAPIContextManagement() && features.AntUserType()
}

// AppendBetaUnique appends name to betas if not already present (order-preserving).
func AppendBetaUnique(betas []string, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return betas
	}
	for _, x := range betas {
		if strings.TrimSpace(x) == name {
			return betas
		}
	}
	return append(append([]string(nil), betas...), name)
}
