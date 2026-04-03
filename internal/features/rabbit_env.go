package features

import (
	"os"
	"strconv"
	"strings"
)

// --- API / HTTP client env (SOURCE_FEATURE_FLAGS, client.ts, withRetry.ts; rabbit-code prefixed) ---

const (
	EnvUnattendedRetry = "RABBIT_CODE_UNATTENDED_RETRY"
	// EnvFastRetry shortens HTTP retry backoff (withRetry.ts fast path); mirrors CLAUDE_CODE_FAST_RETRY when set.
	EnvFastRetry            = "RABBIT_CODE_FAST_RETRY"
	EnvAdditionalProtection = "RABBIT_CODE_ADDITIONAL_PROTECTION"
	EnvUseBedrock           = "RABBIT_CODE_USE_BEDROCK"
	EnvUseVertex            = "RABBIT_CODE_USE_VERTEX"
	EnvUseFoundry           = "RABBIT_CODE_USE_FOUNDRY"
	EnvSkipBedrockAuth      = "RABBIT_CODE_SKIP_BEDROCK_AUTH"
	EnvSkipVertexAuth       = "RABBIT_CODE_SKIP_VERTEX_AUTH"
	EnvSkipFoundryAuth      = "RABBIT_CODE_SKIP_FOUNDRY_AUTH"
	EnvAntiDistillation     = "RABBIT_CODE_ANTI_DISTILLATION_CC"
	// EnvAntiDistillationHeader overrides the HTTP header name when anti-distillation is on (default x-rabbit-code-anti-distillation).
	EnvAntiDistillationHeader = "RABBIT_CODE_ANTI_DISTILLATION_HEADER"
	// EnvAntiDistillationValue sets the header value (default 1).
	EnvAntiDistillationValue = "RABBIT_CODE_ANTI_DISTILLATION_VALUE"
	// EnvAntiDistillationFakeTools when truthy with ANTI_DISTILLATION_CC adds JSON anti_distillation: ["fake_tools"] (claude.ts getExtraBodyParams).
	EnvAntiDistillationFakeTools = "RABBIT_CODE_ANTI_DISTILLATION_FAKE_TOOLS"
	// EnvOAuthBetaAppend is a comma-separated list of extra anthropic-beta names for OAuth sessions (constants/oauth.ts OAUTH_BETA_HEADER patterns).
	EnvOAuthBetaAppend   = "RABBIT_CODE_OAUTH_BETA_APPEND"
	EnvNativeAttestation = "RABBIT_CODE_NATIVE_CLIENT_ATTESTATION"
	// EnvNativeAttestationHeader overrides the HTTP header name when native attestation is on.
	EnvNativeAttestationHeader = "RABBIT_CODE_NATIVE_ATTESTATION_HEADER"
	// EnvNativeAttestationValue sets the header value (default 1).
	EnvNativeAttestationValue = "RABBIT_CODE_NATIVE_ATTESTATION_VALUE"
	EnvPromptCacheBreak       = "RABBIT_CODE_PROMPT_CACHE_BREAK_DETECTION"
	EnvE2EMockAPI             = "RABBIT_CODE_E2E_MOCK_API"
	// EnvOAuthBaseURL overrides console base for usage.ts fetchUtilization (default https://console.anthropic.com).
	EnvOAuthBaseURL = "RABBIT_CODE_OAUTH_BASE_URL"
	// EnvHTTPUserAgent overrides default User-Agent for API HTTP clients (utils/http.ts).
	EnvHTTPUserAgent = "RABBIT_CODE_USER_AGENT"
	// EnvStrictForeground529 sets DefaultPolicy().StrictForeground529 (withRetry.ts FOREGROUND_529_RETRY_SOURCES gate for HTTP 529).
	EnvStrictForeground529 = "RABBIT_CODE_STRICT_FOREGROUND_529"
	// EnvAttributionHeader when set to a falsy value disables the billing attribution system line (CLAUDE_CODE_ATTRIBUTION_HEADER); unset = enabled.
	EnvAttributionHeader = "RABBIT_CODE_ATTRIBUTION_HEADER"
	// EnvDisableKeepAliveOnECONNRESET when truthy wraps *http.Transport to set DisableKeepAlives after ECONNRESET/EPIPE (proxy.ts disableKeepAlive + withRetry.ts stale socket path).
	EnvDisableKeepAliveOnECONNRESET = "RABBIT_CODE_DISABLE_KEEPALIVE_ON_ECONNRESET"
)

// UnattendedRetryEnabled mirrors UNATTENDED_RETRY + CLAUDE_CODE_UNATTENDED_RETRY.
func UnattendedRetryEnabled() bool {
	return truthy(os.Getenv(EnvUnattendedRetry))
}

// FastRetryEnabled mirrors withRetry.ts fast-mode backoff when CLAUDE_CODE_FAST_RETRY / RABBIT_CODE_FAST_RETRY is set.
func FastRetryEnabled() bool {
	return truthy(os.Getenv(EnvFastRetry))
}

// AdditionalProtectionHeader mirrors CLAUDE_CODE_ADDITIONAL_PROTECTION → x-anthropic-additional-protection.
func AdditionalProtectionHeader() bool {
	return truthy(os.Getenv(EnvAdditionalProtection))
}

func UseBedrock() bool         { return truthy(os.Getenv(EnvUseBedrock)) }
func UseVertex() bool          { return truthy(os.Getenv(EnvUseVertex)) }
func UseFoundry() bool         { return truthy(os.Getenv(EnvUseFoundry)) }
func SkipBedrockAuth() bool    { return truthy(os.Getenv(EnvSkipBedrockAuth)) }
func SkipVertexAuth() bool     { return truthy(os.Getenv(EnvSkipVertexAuth)) }
func SkipFoundryAuth() bool    { return truthy(os.Getenv(EnvSkipFoundryAuth)) }
func AntiDistillationCC() bool { return truthy(os.Getenv(EnvAntiDistillation)) }

// AntiDistillationFakeToolsInBody mirrors getExtraBodyParams anti_distillation: opt-in body field when CC is on.
func AntiDistillationFakeToolsInBody() bool {
	return AntiDistillationCC() && truthy(os.Getenv(EnvAntiDistillationFakeTools))
}

// AntiDistillationRequestHeader returns the header name/value to send when ANTI_DISTILLATION_CC is enabled.
func AntiDistillationRequestHeader() (name, value string, ok bool) {
	if !AntiDistillationCC() {
		return "", "", false
	}
	name = strings.TrimSpace(os.Getenv(EnvAntiDistillationHeader))
	if name == "" {
		name = "x-rabbit-code-anti-distillation"
	}
	value = strings.TrimSpace(os.Getenv(EnvAntiDistillationValue))
	if value == "" {
		value = "1"
	}
	return name, value, true
}

// OAuthBetaAppendNames returns extra beta header tokens from RABBIT_CODE_OAUTH_BETA_APPEND (comma-separated).
func OAuthBetaAppendNames() []string {
	s := strings.TrimSpace(os.Getenv(EnvOAuthBetaAppend))
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func NativeClientAttestation() bool {
	return truthy(os.Getenv(EnvNativeAttestation))
}

// NativeAttestationRequestHeader returns the header name/value when NATIVE_CLIENT_ATTESTATION is enabled.
func NativeAttestationRequestHeader() (name, value string, ok bool) {
	if !NativeClientAttestation() {
		return "", "", false
	}
	name = strings.TrimSpace(os.Getenv(EnvNativeAttestationHeader))
	if name == "" {
		name = "x-rabbit-code-client-attestation"
	}
	value = strings.TrimSpace(os.Getenv(EnvNativeAttestationValue))
	if value == "" {
		value = "1"
	}
	return name, value, true
}

func PromptCacheBreakDetection() bool {
	return truthy(os.Getenv(EnvPromptCacheBreak))
}

func E2EMockAPI() bool {
	return truthy(os.Getenv(EnvE2EMockAPI))
}

// StrictForeground529Enabled when true, DefaultPolicy uses strict 529 retry whitelist (see anthropic.foreground529RetrySources).
func StrictForeground529Enabled() bool {
	return truthy(os.Getenv(EnvStrictForeground529))
}

// DisableKeepAliveOnECONNRESETEnabled gates Client transport wrapping (see anthropic.keepAliveResetTransport).
func DisableKeepAliveOnECONNRESETEnabled() bool {
	return truthy(os.Getenv(EnvDisableKeepAliveOnECONNRESET))
}

// AttributionHeaderPromptEnabled mirrors system.ts isAttributionHeaderEnabled: default true unless RABBIT_CODE_ATTRIBUTION_HEADER is set and falsy.
func AttributionHeaderPromptEnabled() bool {
	v, ok := os.LookupEnv(EnvAttributionHeader)
	if !ok {
		return true
	}
	return truthy(v)
}

// --- Headless query / engine env (SOURCE_FEATURE_FLAGS.md P5.F.*; defaults off) ---

const (
	EnvTokenBudget                   = "RABBIT_CODE_TOKEN_BUDGET"
	EnvTokenBudgetMaxInputBytes      = "RABBIT_CODE_TOKEN_BUDGET_MAX_INPUT_BYTES"
	EnvTokenBudgetMaxInputTokens     = "RABBIT_CODE_TOKEN_BUDGET_MAX_INPUT_TOKENS"
	EnvTokenBudgetMaxAttachmentBytes = "RABBIT_CODE_TOKEN_BUDGET_MAX_ATTACHMENT_BYTES"
	// EnvTokenSubmitEstimateMode selects how resolved Submit text is tokenized for MAX_INPUT_TOKENS (H5): "bytes4" (default), "structured" (Messages JSON array), or "api" (Anthropic count_tokens with heuristic fallback).
	EnvTokenSubmitEstimateMode = "RABBIT_CODE_TOKEN_SUBMIT_ESTIMATE_MODE"
	EnvReactiveCompact         = "RABBIT_CODE_REACTIVE_COMPACT"
	// EnvTenguCobaltRaccoon mirrors GrowthBook tengu_cobalt_raccoon (reactive-only mode under REACTIVE_COMPACT).
	EnvTenguCobaltRaccoon             = "RABBIT_CODE_TENGU_COBALT_RACCOON"
	EnvContextCollapse                = "RABBIT_CODE_CONTEXT_COLLAPSE"
	EnvSessionRestore                 = "RABBIT_CODE_SESSION_RESTORE"
	EnvUltrathink                     = "RABBIT_CODE_ULTRATHINK"
	EnvUltraplan                      = "RABBIT_CODE_ULTRAPLAN"
	EnvBreakCacheCommand              = "RABBIT_CODE_BREAK_CACHE_COMMAND"
	EnvTemplates                      = "RABBIT_CODE_TEMPLATES"
	EnvCachedMicrocompact             = "RABBIT_CODE_CACHED_MICROCOMPACT"
	EnvHistorySnip                    = "RABBIT_CODE_HISTORY_SNIP"
	EnvBashExec                       = "RABBIT_CODE_BASH_EXEC"
	EnvSnipCompact                    = "RABBIT_CODE_SNIP_COMPACT"
	EnvSnipCompactMaxBytes            = "RABBIT_CODE_SNIP_COMPACT_MAX_BYTES"
	EnvSnipCompactMaxRounds           = "RABBIT_CODE_SNIP_COMPACT_MAX_ROUNDS"
	EnvReactiveCompactMinBytes        = "RABBIT_CODE_REACTIVE_COMPACT_MIN_BYTES"
	EnvReactiveCompactMinTokens       = "RABBIT_CODE_REACTIVE_COMPACT_MIN_TOKENS"
	EnvHistorySnipMaxBytes            = "RABBIT_CODE_HISTORY_SNIP_MAX_BYTES"
	EnvHistorySnipMaxRounds           = "RABBIT_CODE_HISTORY_SNIP_MAX_ROUNDS"
	EnvTemplateNames                  = "RABBIT_CODE_TEMPLATE_NAMES"
	EnvTemplateDir                    = "RABBIT_CODE_TEMPLATE_DIR"
	EnvPromptCacheBreakSuggestCompact = "RABBIT_CODE_PROMPT_CACHE_BREAK_SUGGEST_COMPACT"
	EnvPromptCacheBreakTrimResend     = "RABBIT_CODE_PROMPT_CACHE_BREAK_TRIM_RESEND"
	EnvPromptCacheBreakAutoCompact    = "RABBIT_CODE_PROMPT_CACHE_BREAK_AUTO_COMPACT"
	// Autocompact / proactive compact (autoCompact.ts + isAutoCompactEnabled; rabbit-code prefixed).
	EnvDisableCompact               = "RABBIT_CODE_DISABLE_COMPACT"
	EnvDisableAutoCompact           = "RABBIT_CODE_DISABLE_AUTO_COMPACT"
	EnvAutoCompact                  = "RABBIT_CODE_AUTO_COMPACT" // unset = on; "0"/"false" = user off
	EnvSuppressProactiveAutoCompact = "RABBIT_CODE_SUPPRESS_PROACTIVE_AUTO_COMPACT"
	EnvContextWindowTokens          = "RABBIT_CODE_CONTEXT_WINDOW_TOKENS"
	EnvAutoCompactWindow            = "RABBIT_CODE_AUTO_COMPACT_WINDOW"
	EnvAutocompactPctOverride       = "RABBIT_CODE_AUTOCOMPACT_PCT_OVERRIDE"
	EnvBlockingLimitOverride        = "RABBIT_CODE_BLOCKING_LIMIT_OVERRIDE"
	// EnvMemdirRelevanceMode selects memdir memory selection: "heuristic" (default) or "llm" (side-query, H8).
	EnvMemdirRelevanceMode = "RABBIT_CODE_MEMDIR_RELEVANCE_MODE"
	// EnvMemdirStrictLLM when truthy, LLM memdir selection errors yield no memories (no heuristic fallback; TS-aligned).
	EnvMemdirStrictLLM = "RABBIT_CODE_MEMDIR_STRICT_LLM"
	// EnvMemdirMemoryDir sets the engine memory scan directory when Config.MemdirMemoryDir is empty (H8).
	EnvMemdirMemoryDir = "RABBIT_CODE_MEMDIR_MEMORY_DIR"
	// EnvAutoMemdir when truthy, enables default layout resolution when Config.MemdirMemoryDir and EnvMemdirMemoryDir are unset (requires AutoMemoryEnabled).
	// Config.MemdirTrustedAutoMemoryDirectory alone (from config.LoadTrustedAutoMemoryDirectory) can enable auto-resolve without this env.
	EnvAutoMemdir = "RABBIT_CODE_AUTO_MEMDIR"
	// Auto-memory gates (memdir/paths.ts isAutoMemoryEnabled); use AutoMemoryEnabledFromMerged with config.LoadMerged for autoMemoryEnabled.
	EnvDisableAutoMemory       = "RABBIT_CODE_DISABLE_AUTO_MEMORY"
	EnvClaudeDisableAutoMemory = "CLAUDE_CODE_DISABLE_AUTO_MEMORY"
	EnvSimple                  = "RABBIT_CODE_SIMPLE"
	EnvClaudeSimple            = "CLAUDE_CODE_SIMPLE"
	EnvRemote                  = "RABBIT_CODE_REMOTE"
	EnvClaudeRemote            = "CLAUDE_CODE_REMOTE"
	EnvRemoteMemoryDir         = "RABBIT_CODE_REMOTE_MEMORY_DIR"
	EnvClaudeRemoteMemoryDir   = "CLAUDE_CODE_REMOTE_MEMORY_DIR"
)

// MemdirRelevanceMode returns memdir.RelevanceMode values: "heuristic" or "llm".
func MemdirRelevanceMode() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(EnvMemdirRelevanceMode)))
	switch v {
	case "llm", "side_query", "side-query":
		return "llm"
	default:
		return "heuristic"
	}
}

// MemdirStrictLLM mirrors strict LLM-only recall (findRelevantMemories.ts returns [] on sideQuery failure).
func MemdirStrictLLM() bool {
	return truthy(os.Getenv(EnvMemdirStrictLLM))
}

// MemdirMemoryDirFromEnv returns RABBIT_CODE_MEMDIR_MEMORY_DIR trimmed.
func MemdirMemoryDirFromEnv() string {
	return strings.TrimSpace(os.Getenv(EnvMemdirMemoryDir))
}

// AutoMemdirFromProject is true when RABBIT_CODE_AUTO_MEMDIR is truthy.
func AutoMemdirFromProject() bool {
	return truthy(os.Getenv(EnvAutoMemdir))
}

func envDefinedDual(rabbitKey, claudeKey string) (string, bool) {
	if v, ok := os.LookupEnv(rabbitKey); ok {
		return v, true
	}
	if v, ok := os.LookupEnv(claudeKey); ok {
		return v, true
	}
	return "", false
}

func firstNonEmptyEnvPair(rabbitKey, claudeKey string) string {
	if s := strings.TrimSpace(os.Getenv(rabbitKey)); s != "" {
		return s
	}
	return os.Getenv(claudeKey)
}

func explicitFalsyEnv(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "0", "false", "no", "off":
		return true
	}
	if b, err := strconv.ParseBool(s); err == nil && !b {
		return true
	}
	return false
}

// AutoMemoryEnabled mirrors paths.ts isAutoMemoryEnabled without merged settings (same as AutoMemoryEnabledFromMerged(nil)).
func AutoMemoryEnabled() bool {
	return AutoMemoryEnabledFromMerged(nil)
}

// AutoMemoryEnabledFromMerged mirrors paths.ts isAutoMemoryEnabled when merged includes project settings (e.g. config.LoadMerged).
// After env gates, if merged contains key autoMemoryEnabled and the value parses as a boolean, that result wins; otherwise default true.
func AutoMemoryEnabledFromMerged(merged map[string]interface{}) bool {
	v, ok := envDefinedDual(EnvDisableAutoMemory, EnvClaudeDisableAutoMemory)
	if ok {
		if truthy(v) {
			return false
		}
		if explicitFalsyEnv(v) {
			return true
		}
	}
	if truthy(firstNonEmptyEnvPair(EnvSimple, EnvClaudeSimple)) {
		return false
	}
	if truthy(firstNonEmptyEnvPair(EnvRemote, EnvClaudeRemote)) {
		md := strings.TrimSpace(firstNonEmptyEnvPair(EnvRemoteMemoryDir, EnvClaudeRemoteMemoryDir))
		if md == "" {
			return false
		}
	}
	if merged != nil {
		raw, exists := merged["autoMemoryEnabled"]
		if exists {
			if b, ok := parseJSONBoolSetting(raw); ok {
				return b
			}
		}
	}
	return true
}

func parseJSONBoolSetting(v interface{}) (bool, bool) {
	if v == nil {
		return false, false
	}
	switch x := v.(type) {
	case bool:
		return x, true
	case float64:
		if x == 0 {
			return false, true
		}
		if x == 1 {
			return true, true
		}
		return false, false
	case string:
		s := strings.TrimSpace(strings.ToLower(x))
		switch s {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
		if b, err := strconv.ParseBool(s); err == nil {
			return b, true
		}
		return false, false
	default:
		return false, false
	}
}

func TokenBudgetEnabled() bool { return truthy(os.Getenv(EnvTokenBudget)) }

// TokenBudgetMaxInputBytes enforces a UTF-8 byte cap on resolved Submit text when TOKEN_BUDGET is on.
func TokenBudgetMaxInputBytes() int {
	if !TokenBudgetEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvTokenBudgetMaxInputBytes))
	if s == "" {
		return 4_000_000
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 4_000_000
	}
	return v
}

func TokenBudgetMaxInputTokens() int {
	if !TokenBudgetEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvTokenBudgetMaxInputTokens))
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func TokenBudgetMaxAttachmentBytes() int {
	if !TokenBudgetEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvTokenBudgetMaxAttachmentBytes))
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// SubmitTokenEstimateMode returns "structured", "api", or "bytes4" for submit input token estimates (only meaningful when TOKEN_BUDGET is on).
func SubmitTokenEstimateMode() string {
	if !TokenBudgetEnabled() {
		return "bytes4"
	}
	s := strings.ToLower(strings.TrimSpace(os.Getenv(EnvTokenSubmitEstimateMode)))
	switch s {
	case "structured", "api":
		return s
	default:
		return "bytes4"
	}
}

func ReactiveCompactEnabled() bool { return truthy(os.Getenv(EnvReactiveCompact)) }

// TenguCobaltRaccoon is headless stand-in for getFeatureValue_CACHED_MAY_BE_STALE('tengu_cobalt_raccoon').
// When ReactiveCompactEnabled and this is true, shouldAutoCompact suppresses proactive autocompact (autoCompact.ts).
func TenguCobaltRaccoon() bool { return truthy(os.Getenv(EnvTenguCobaltRaccoon)) }

func ContextCollapseEnabled() bool { return truthy(os.Getenv(EnvContextCollapse)) }

// DisableCompact mirrors DISABLE_COMPACT (blocks autocompact entry in autoCompact.ts).
func DisableCompact() bool { return truthy(os.Getenv(EnvDisableCompact)) }

// DisableAutoCompact mirrors DISABLE_AUTO_COMPACT.
func DisableAutoCompact() bool { return truthy(os.Getenv(EnvDisableAutoCompact)) }

// AutoCompactUserPreferenceEnabled mirrors userConfig.autoCompactEnabled when env is unset (default on).
// Empty string is treated like unset so t.Setenv(EnvAutoCompact, "") does not disable autocompact.
func AutoCompactUserPreferenceEnabled() bool {
	v, ok := os.LookupEnv(EnvAutoCompact)
	if !ok {
		return true
	}
	s := strings.TrimSpace(v)
	if s == "" {
		return true
	}
	return truthy(s)
}

// IsAutoCompactEnabled mirrors isAutoCompactEnabled() in autoCompact.ts.
func IsAutoCompactEnabled() bool {
	if DisableCompact() {
		return false
	}
	if DisableAutoCompact() {
		return false
	}
	return AutoCompactUserPreferenceEnabled()
}

// SuppressProactiveAutoCompact mirrors reactive-only / context-collapse style suppression of shouldAutoCompact
// (tengu_cobalt_raccoon is a separate GrowthBook flag in TS; headless uses RABBIT_CODE_SUPPRESS_PROACTIVE_AUTO_COMPACT).
func SuppressProactiveAutoCompact() bool {
	return truthy(os.Getenv(EnvSuppressProactiveAutoCompact))
}

func defaultContextWindowForModel(model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(m, "1m") || strings.Contains(m, "1000000") {
		return 1_000_000
	}
	return 200_000
}

// ContextWindowTokensForModel returns RABBIT_CODE_CONTEXT_WINDOW_TOKENS or a heuristic default (200k / 1M).
// Does not apply RABBIT_CODE_AUTO_COMPACT_WINDOW — use ApplyAutoCompactWindowCap after this.
func ContextWindowTokensForModel(model string) int {
	s := strings.TrimSpace(os.Getenv(EnvContextWindowTokens))
	if s != "" {
		v, err := strconv.Atoi(s)
		if err == nil && v > 0 {
			return v
		}
	}
	return defaultContextWindowForModel(model)
}

// ApplyAutoCompactWindowCap mirrors CLAUDE_CODE_AUTO_COMPACT_WINDOW (caps context window from above).
func ApplyAutoCompactWindowCap(contextWindow int) int {
	if contextWindow <= 0 {
		return contextWindow
	}
	s := strings.TrimSpace(os.Getenv(EnvAutoCompactWindow))
	if s == "" {
		return contextWindow
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return contextWindow
	}
	if v < contextWindow {
		return v
	}
	return contextWindow
}

// AutocompactPctOverride mirrors CLAUDE_AUTOCOMPACT_PCT_OVERRIDE (0 = unset).
func AutocompactPctOverride() float64 {
	s := strings.TrimSpace(os.Getenv(EnvAutocompactPctOverride))
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 || v > 100 {
		return 0
	}
	return v
}

// BlockingLimitOverrideTokens mirrors CLAUDE_CODE_BLOCKING_LIMIT_OVERRIDE for calculateTokenWarningState (0 = use default).
func BlockingLimitOverrideTokens() int {
	s := strings.TrimSpace(os.Getenv(EnvBlockingLimitOverride))
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}
func SessionRestoreEnabled() bool     { return truthy(os.Getenv(EnvSessionRestore)) }
func UltrathinkEnabled() bool         { return truthy(os.Getenv(EnvUltrathink)) }
func UltraplanEnabled() bool          { return truthy(os.Getenv(EnvUltraplan)) }
func BreakCacheCommandEnabled() bool  { return truthy(os.Getenv(EnvBreakCacheCommand)) }
func TemplatesEnabled() bool          { return truthy(os.Getenv(EnvTemplates)) }
func CachedMicrocompactEnabled() bool { return truthy(os.Getenv(EnvCachedMicrocompact)) }
func HistorySnipEnabled() bool        { return truthy(os.Getenv(EnvHistorySnip)) }
func SnipCompactEnabled() bool        { return truthy(os.Getenv(EnvSnipCompact)) }
func BashExecEnabled() bool           { return truthy(os.Getenv(EnvBashExec)) }

func SnipCompactMaxBytes() int {
	if !SnipCompactEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvSnipCompactMaxBytes))
	if s == "" {
		return 32768
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 32768
	}
	return v
}

func SnipCompactMaxRounds() int {
	if !SnipCompactEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvSnipCompactMaxRounds))
	if s == "" {
		return 4
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 4
	}
	return v
}

// PromptCacheBreakDetectionEnabled aliases PROMPT_CACHE_BREAK_DETECTION (shared with anthropic stream path).
func PromptCacheBreakDetectionEnabled() bool { return PromptCacheBreakDetection() }

func PromptCacheBreakSuggestCompactEnabled() bool {
	return truthy(os.Getenv(EnvPromptCacheBreakSuggestCompact))
}

func PromptCacheBreakTrimResendEnabled() bool {
	if !PromptCacheBreakDetection() {
		return false
	}
	v := strings.TrimSpace(os.Getenv(EnvPromptCacheBreakTrimResend))
	if v == "" {
		return true
	}
	return truthy(v)
}

func PromptCacheBreakAutoCompactEnabled() bool {
	if !PromptCacheBreakDetection() {
		return false
	}
	return truthy(os.Getenv(EnvPromptCacheBreakAutoCompact))
}

func ReactiveCompactMinTranscriptBytes() int {
	if !ReactiveCompactEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvReactiveCompactMinBytes))
	if s == "" {
		return 8192
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 8192
	}
	return v
}

func ReactiveCompactMinEstimatedTokens() int {
	if !ReactiveCompactEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvReactiveCompactMinTokens))
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func HistorySnipMaxBytes() int {
	if !HistorySnipEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvHistorySnipMaxBytes))
	if s == "" {
		return 32768
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 32768
	}
	return v
}

func HistorySnipMaxRounds() int {
	if !HistorySnipEnabled() {
		return 0
	}
	s := strings.TrimSpace(os.Getenv(EnvHistorySnipMaxRounds))
	if s == "" {
		return 4
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 4
	}
	return v
}

func TemplateNames() []string {
	if !TemplatesEnabled() {
		return nil
	}
	return splitCommaEnv(os.Getenv(EnvTemplateNames))
}

func TemplateMarkdownDir() string {
	if !TemplatesEnabled() {
		return ""
	}
	return strings.TrimSpace(os.Getenv(EnvTemplateDir))
}

func splitCommaEnv(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
