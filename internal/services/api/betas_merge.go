package anthropic

import (
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// DedupeBetasPreserveOrder removes empty and duplicate beta names, keeping first occurrence (utils/betas.ts getMergedBetas merge).
func DedupeBetasPreserveOrder(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	var out []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// SplitBetasForBedrock returns betas that belong in HTTP anthropic-beta vs Bedrock extraBodyParams
// (constants/betas.ts BEDROCK_EXTRA_PARAMS_HEADERS).
func SplitBetasForBedrock(names []string) (header []string, extraBody []string) {
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := BedrockExtraParamsBetas[n]; ok {
			extraBody = append(extraBody, n)
		} else {
			header = append(header, n)
		}
	}
	return header, extraBody
}

// MergeBetasForProvider builds the anthropic-beta header value for HTTP.
// For Bedrock, betas in BedrockExtraParamsBetas are omitted here (caller merges them into JSON body).
func MergeBetasForProvider(p Provider, names []string) string {
	names = DedupeBetasPreserveOrder(names)
	if p == ProviderBedrock {
		h, _ := SplitBetasForBedrock(names)
		return MergeBetaHeader(h)
	}
	return MergeBetaHeader(names)
}

// VertexCountTokensAllowed returns betas permitted on Vertex countTokens (constants/betas.ts VERTEX_COUNT_TOKENS_ALLOWED_BETAS).
var VertexCountTokensAllowed = map[string]struct{}{
	BetaClaudeCode20250219:  {},
	BetaInterleavedThinking: {},
	BetaContextManagement:   {},
}

// FilterBetasVertexCountTokens keeps only allowed betas for count-tokens style calls.
// MergeBetaHeaderAppend adds betaName to a comma-separated anthropic-beta value if not already present.
func MergeBetaHeaderAppend(headerValue, betaName string) string {
	betaName = strings.TrimSpace(betaName)
	if betaName == "" {
		return headerValue
	}
	for _, p := range strings.Split(headerValue, ",") {
		if strings.TrimSpace(p) == betaName {
			return headerValue
		}
	}
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return betaName
	}
	return headerValue + "," + betaName
}

func FilterBetasVertexCountTokens(names []string) []string {
	var out []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := VertexCountTokensAllowed[n]; ok {
			out = append(out, n)
		}
	}
	return out
}

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
