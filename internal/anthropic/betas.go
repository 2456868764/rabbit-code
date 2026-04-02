package anthropic

// Beta header name strings aligned with constants/betas.ts (Phase 4 / P4.F.6).
const (
	BetaClaudeCode20250219     = "claude-code-20250219"
	BetaInterleavedThinking    = "interleaved-thinking-2025-05-14"
	BetaContext1M              = "context-1m-2025-08-07"
	BetaContextManagement      = "context-management-2025-06-27"
	BetaStructuredOutputs      = "structured-outputs-2025-12-15"
	BetaWebSearch              = "web-search-2025-03-05"
	BetaToolSearch1P           = "advanced-tool-use-2025-11-20"
	BetaToolSearch3P           = "tool-search-tool-2025-10-19"
	BetaEffort                 = "effort-2025-11-24"
	BetaTaskBudgets            = "task-budgets-2026-03-13"
	BetaPromptCachingScope     = "prompt-caching-scope-2026-01-05"
	BetaFastMode               = "fast-mode-2026-02-01"
	BetaRedactThinking         = "redact-thinking-2026-02-12"
	BetaTokenEfficientTools    = "token-efficient-tools-2026-03-28"
	BetaSummarizeConnectorText = "summarize-connector-text-2026-03-13"
	BetaAFKMode                = "afk-mode-2026-01-31"
	BetaAdvisor                = "advisor-tool-2026-03-01"
	// BetaCLIInternal is constants/betas.ts CLI_INTERNAL_BETA_HEADER when USER_TYPE=ant (ant-only in product; string frozen for parity).
	BetaCLIInternal = "cli-internal-2026-02-09"
	// BetaOAuth is constants/oauth.ts OAUTH_BETA_HEADER (merged into anthropic-beta on OAuth-backed clients).
	BetaOAuth = "oauth-2025-04-20"
)

// MergeBetaHeader joins non-empty beta names for the anthropic-beta HTTP header (utils/betas.ts getMergedBetas).
func MergeBetaHeader(names []string) string {
	var b []byte
	for _, n := range names {
		if n == "" {
			continue
		}
		if len(b) > 0 {
			b = append(b, ',')
		}
		b = append(b, n...)
	}
	return string(b)
}

// BedrockExtraParamsBetas is the set that Bedrock carries in extraBodyParams (constants/betas.ts BEDROCK_EXTRA_PARAMS_HEADERS).
var BedrockExtraParamsBetas = map[string]struct{}{
	BetaInterleavedThinking: {},
	BetaContext1M:           {},
	BetaToolSearch3P:        {},
}
