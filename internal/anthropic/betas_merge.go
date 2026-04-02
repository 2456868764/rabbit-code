package anthropic

import (
	"strings"
)

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
