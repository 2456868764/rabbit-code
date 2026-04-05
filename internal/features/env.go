// Package features holds runtime feature flags aligned with SOURCE_FEATURE_FLAGS.md.
// All env-backed toggles live in this file (Phase-1 core, API client, headless query, memdir).
// Later phases may merge with config/telemetry.
package features

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Environment variable names — Phase-1-style core toggles (rabbit-code convention).
const (
	EnvHardFail             = "RABBIT_CODE_HARD_FAIL"
	EnvSlowOperationLogging = "RABBIT_CODE_SLOW_OPERATION_LOGGING"
	EnvFilePersistence      = "RABBIT_CODE_FILE_PERSISTENCE"
	EnvLodestone            = "RABBIT_CODE_LODESTONE"
)

func truthy(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1", "true", "yes", "on":
		return true
	}
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return false
}

// HardFailEnabled mirrors feature('HARD_FAIL'): treat bootstrap failures as fatal-style (non-zero exit, stderr prefix).
func HardFailEnabled() bool {
	return truthy(os.Getenv(EnvHardFail))
}

// SlowOperationLoggingEnabled mirrors feature('SLOW_OPERATION_LOGGING'): separate slow-op log sink when true.
func SlowOperationLoggingEnabled() bool {
	return truthy(os.Getenv(EnvSlowOperationLogging))
}

// FilePersistenceEnabled mirrors feature('FILE_PERSISTENCE'): register startup/shutdown hooks (Phase 1 no-op body).
func FilePersistenceEnabled() bool {
	return truthy(os.Getenv(EnvFilePersistence))
}

// LodestoneEnabled mirrors feature('LODESTONE'): interactive-only bootstrap hook until Phase 12 registers protocols.
func LodestoneEnabled() bool {
	return truthy(os.Getenv(EnvLodestone))
}

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
	// DefaultHTTPUserAgent is used when EnvHTTPUserAgent is unset (rabbit-code/api).
	DefaultHTTPUserAgent = "rabbit-code/api"
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

// HTTPUserAgent returns the app HTTP User-Agent string (EnvHTTPUserAgent trimmed, else DefaultHTTPUserAgent).
// Used by Anthropic API clients and WebFetch (utils/http.ts getUserAgent parity).
func HTTPUserAgent() string {
	if v := strings.TrimSpace(os.Getenv(EnvHTTPUserAgent)); v != "" {
		return v
	}
	return DefaultHTTPUserAgent
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

// --- API context management (apiMicrocompact.ts; env names match upstream for Anthropic API paths) ---

const (
	EnvUserType               = "USER_TYPE"
	EnvUserTypeRabbit         = "RABBIT_CODE_USER_TYPE"
	EnvUseAPIClearToolResults = "USE_API_CLEAR_TOOL_RESULTS"
	EnvUseAPIClearToolUses    = "USE_API_CLEAR_TOOL_USES"
	// EnvUseAPIContextManagement mirrors USE_API_CONTEXT_MANAGEMENT (betas.ts ant tool-clearing gate for context-management beta).
	EnvUseAPIContextManagement = "USE_API_CONTEXT_MANAGEMENT"
	EnvAPIMaxInputTokens       = "API_MAX_INPUT_TOKENS"
	EnvAPITargetInputTokens    = "API_TARGET_INPUT_TOKENS"
	// EnvSkipWebFetchPreflight skips api.anthropic.com domain_info preflight (settings skipWebFetchPreflight analogue).
	EnvSkipWebFetchPreflight = "RABBIT_CODE_SKIP_WEBFETCH_PREFLIGHT"
)

// AntUserType is true when USER_TYPE or RABBIT_CODE_USER_TYPE equals "ant" (apiMicrocompact ant-only branches).
func AntUserType() bool {
	ut := strings.TrimSpace(os.Getenv(EnvUserTypeRabbit))
	if ut == "" {
		ut = strings.TrimSpace(os.Getenv(EnvUserType))
	}
	return strings.EqualFold(ut, "ant")
}

// UseAPIClearToolResults mirrors isEnvTruthy(USE_API_CLEAR_TOOL_RESULTS).
func UseAPIClearToolResults() bool {
	return truthy(os.Getenv(EnvUseAPIClearToolResults))
}

// UseAPIClearToolUses mirrors isEnvTruthy(USE_API_CLEAR_TOOL_USES).
func UseAPIClearToolUses() bool {
	return truthy(os.Getenv(EnvUseAPIClearToolUses))
}

// UseAPIContextManagement mirrors isEnvTruthy(USE_API_CONTEXT_MANAGEMENT) (ant opt-in for tool clearing + beta).
func UseAPIContextManagement() bool {
	return truthy(os.Getenv(EnvUseAPIContextManagement))
}

// SkipWebFetchPreflight mirrors settings skipWebFetchPreflight (enterprise / no outbound to claude.ai).
func SkipWebFetchPreflight() bool {
	return truthy(os.Getenv(EnvSkipWebFetchPreflight))
}

const (
	// EnvRedactThinking enables compact.APIContextManagementOptions.IsRedactThinkingActive (betas REDACT_THINKING_BETA_HEADER analogue).
	EnvRedactThinking = "RABBIT_CODE_REDACT_THINKING"
	// EnvThinkingClearAll sets ClearAllThinking (apiMicrocompact.ts clearAllThinking / thinkingClearLatched analogue).
	EnvThinkingClearAll = "RABBIT_CODE_THINKING_CLEAR_ALL"
)

// RedactThinkingEnabled is true when RABBIT_CODE_REDACT_THINKING is truthy.
func RedactThinkingEnabled() bool {
	return truthy(os.Getenv(EnvRedactThinking))
}

// ThinkingClearAllLatched is true when RABBIT_CODE_THINKING_CLEAR_ALL is truthy.
func ThinkingClearAllLatched() bool {
	return truthy(os.Getenv(EnvThinkingClearAll))
}

const defaultAPIMaxInputTokens = 180_000
const defaultAPITargetInputTokens = 40_000

// APIMaxInputTokens mirrors API_MAX_INPUT_TOKENS parse with default 180_000.
func APIMaxInputTokens() int {
	s := strings.TrimSpace(os.Getenv(EnvAPIMaxInputTokens))
	if s == "" {
		return defaultAPIMaxInputTokens
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultAPIMaxInputTokens
	}
	return v
}

// APITargetInputTokens mirrors API_TARGET_INPUT_TOKENS parse with default 40_000.
func APITargetInputTokens() int {
	s := strings.TrimSpace(os.Getenv(EnvAPITargetInputTokens))
	if s == "" {
		return defaultAPITargetInputTokens
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultAPITargetInputTokens
	}
	return v
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
	EnvTenguCobaltRaccoon = "RABBIT_CODE_TENGU_COBALT_RACCOON"
	EnvContextCollapse    = "RABBIT_CODE_CONTEXT_COLLAPSE"
	// EnvContextCollapseInactive when truthy: CONTEXT_COLLAPSE env is on but runtime collapse is off — proactive autocompact may run (autoCompact.ts isContextCollapseEnabled false).
	EnvContextCollapseInactive = "RABBIT_CODE_CONTEXT_COLLAPSE_INACTIVE"
	// EnvCommitAttribution mirrors internal COMMIT_ATTRIBUTION (postCompactCleanup.ts sweepFileContentCache gate).
	EnvCommitAttribution = "RABBIT_CODE_COMMIT_ATTRIBUTION"
	EnvSessionRestore    = "RABBIT_CODE_SESSION_RESTORE"
	// EnvClaudeCodeEnableTasks mirrors CLAUDE_CODE_ENABLE_TASKS (tasks.ts): when set, Task v2 is on and TodoWrite is disabled.
	EnvClaudeCodeEnableTasks = "CLAUDE_CODE_ENABLE_TASKS"
	// EnvVerificationAgent mirrors bundle feature('VERIFICATION_AGENT') (builtInAgents.ts / TodoWriteTool nudge gate).
	EnvVerificationAgent = "RABBIT_CODE_VERIFICATION_AGENT"
	// EnvTenguHiveEvidence mirrors getFeatureValue_CACHED_MAY_BE_STALE('tengu_hive_evidence', false) (TodoWriteTool.ts).
	EnvTenguHiveEvidence = "RABBIT_CODE_TENGU_HIVE_EVIDENCE"
	EnvUltrathink        = "RABBIT_CODE_ULTRATHINK"
	EnvUltraplan         = "RABBIT_CODE_ULTRAPLAN"
	EnvBreakCacheCommand = "RABBIT_CODE_BREAK_CACHE_COMMAND"
	EnvTemplates         = "RABBIT_CODE_TEMPLATES"
	// EnvStopHooksDir is a directory of *.md stop-hook scripts (query/stopHooks.ts job dir analogue; list-only CLI in rabbit-code).
	EnvStopHooksDir       = "RABBIT_CODE_STOP_HOOKS_DIR"
	EnvCachedMicrocompact = "RABBIT_CODE_CACHED_MICROCOMPACT"
	// Compact API behaviour (compact.ts streamCompactSummary GrowthBook keys → env).
	EnvCompactStreamingRetry = "RABBIT_CODE_COMPACT_STREAMING_RETRY" // tengu_compact_streaming_retry, default off
	EnvCompactCachePrefix    = "RABBIT_CODE_COMPACT_CACHE_PREFIX"    // tengu_compact_cache_prefix, default on (try ForkCompactSummary when set)
	EnvCompactToolSearch     = "RABBIT_CODE_COMPACT_TOOL_SEARCH"     // include ToolSearch in compact stream tools (default off)
	// EnvRemoteSendKeepalives mirrors CLAUDE_CODE_REMOTE_SEND_KEEPALIVES (sessionActivity.ts sendSessionActivitySignal gate).
	EnvRemoteSendKeepalives       = "CLAUDE_CODE_REMOTE_SEND_KEEPALIVES"
	EnvRemoteSendKeepalivesRabbit = "RABBIT_CODE_REMOTE_SEND_KEEPALIVES"
	// Time-based microcompact (timeBasedMCConfig.ts; GrowthBook key tengu_slate_heron → env analogue).
	EnvTimeBasedMicrocompact          = "RABBIT_CODE_TIME_BASED_MICROCOMPACT"
	EnvTimeBasedMCGapMinutes          = "RABBIT_CODE_TIME_BASED_MC_GAP_MINUTES"
	EnvTimeBasedMCKeepRecent          = "RABBIT_CODE_TIME_BASED_MC_KEEP_RECENT"
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
	// EnvRabbitCodeConfigDir overrides Claude config home (~/.claude); memdir/paths.ts getClaudeConfigHomeDir.
	EnvRabbitCodeConfigDir = "RABBIT_CODE_CONFIG_DIR"
	EnvClaudeConfigDir     = "CLAUDE_CONFIG_DIR"
	// EnvTeamMem enables team memory (TEAMMEM build + GrowthBook tengu_herring_clock in TS). Headless: default off unless truthy.
	EnvTeamMem = "RABBIT_CODE_TEAMMEM"
	// EnvMemorySearchPastContext enables "## Searching past context" in memory prompts (memdir.ts buildSearchingPastContextSection).
	EnvMemorySearchPastContext = "RABBIT_CODE_MEMORY_SEARCH_PAST_CONTEXT"
	// EnvExtractMemories enables forked background memory extraction after successful Submit (stopHooks.ts + extractMemories.ts).
	EnvExtractMemories = "RABBIT_CODE_EXTRACT_MEMORIES"
	// EnvExtractMemoriesNonInteractive allows extract when the session is non-interactive (paths.ts tengu_slate_thimble analogue).
	EnvExtractMemoriesNonInteractive = "RABBIT_CODE_EXTRACT_MEMORIES_NON_INTERACTIVE"
	// EnvExtractMemoriesInterval runs extraction at most every N eligible stop-hook invocations (default 1). TS: tengu_bramble_lintel.
	EnvExtractMemoriesInterval = "RABBIT_CODE_EXTRACT_MEMORIES_INTERVAL"
	// EnvExtractMemoriesSkipIndex when truthy passes skipIndex to extract prompts (TS: tengu_moth_copse).
	EnvExtractMemoriesSkipIndex = "RABBIT_CODE_EXTRACT_MEMORIES_SKIP_INDEX"
	// EnvMemorySystemPrompt when unset, memory system prompt injection is on when auto memory resolves (H8). Set to "0" to disable.
	EnvMemorySystemPrompt = "RABBIT_CODE_MEMORY_SYSTEM_PROMPT"
	// EnvKairosDailyLogMemory + EnvKairosActive enable assistant daily-log memory mode (memdir.ts KAIROS branch).
	EnvKairosDailyLogMemory = "RABBIT_CODE_KAIROS_DAILY_LOG_MEMORY"
	EnvKairosActive         = "RABBIT_CODE_KAIROS_ACTIVE"
	// EnvCoworkMemoryExtraGuidelines appends lines to memory builders (CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES analogue).
	EnvCoworkMemoryExtraGuidelines = "RABBIT_CODE_COWORK_MEMORY_EXTRA_GUIDELINES"
	// EnvExperimentalSkillSearch mirrors feature('EXPERIMENTAL_SKILL_SEARCH') (compact.ts stripReinjectedAttachments).
	EnvExperimentalSkillSearch = "RABBIT_CODE_EXPERIMENTAL_SKILL_SEARCH"
	// Session memory compaction (sessionMemoryCompact.ts shouldUseSessionMemoryCompaction + GrowthBook tengu_session_memory / tengu_sm_compact).
	EnvEnableClaudeCodeSMCompact   = "ENABLE_CLAUDE_CODE_SM_COMPACT"      // mirrors ENABLE_CLAUDE_CODE_SM_COMPACT
	EnvDisableClaudeCodeSMCompact  = "DISABLE_CLAUDE_CODE_SM_COMPACT"     // mirrors DISABLE_CLAUDE_CODE_SM_COMPACT
	EnvSessionMemoryFeature        = "RABBIT_CODE_SESSION_MEMORY"         // tengu_session_memory
	EnvSessionMemoryCompactFeature = "RABBIT_CODE_SESSION_MEMORY_COMPACT" // tengu_sm_compact
	EnvSMCompactMinTokens          = "RABBIT_CODE_SM_COMPACT_MIN_TOKENS"
	EnvSMCompactMinTextMessages    = "RABBIT_CODE_SM_COMPACT_MIN_TEXT_MESSAGES"
	EnvSMCompactMaxTokens          = "RABBIT_CODE_SM_COMPACT_MAX_TOKENS"
	// Cached microcompact thresholds (cachedMicrocompact.ts GrowthBook analogue).
	EnvCachedMCTriggerThreshold = "RABBIT_CODE_CACHED_MC_TRIGGER_THRESHOLD"
	EnvCachedMCKeepRecent       = "RABBIT_CODE_CACHED_MC_KEEP_RECENT"
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

// ConfigHomeDir resolves Claude-style config home (envUtils.getClaudeConfigHomeDir): RABBIT_CODE_CONFIG_DIR, CLAUDE_CONFIG_DIR, else ~/.claude.
func ConfigHomeDir() string {
	if s := strings.TrimSpace(os.Getenv(EnvRabbitCodeConfigDir)); s != "" {
		return filepath.Clean(s)
	}
	if s := strings.TrimSpace(os.Getenv(EnvClaudeConfigDir)); s != "" {
		return filepath.Clean(s)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
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

// TeamMemoryEnabled mirrors teamMemPaths.isTeamMemoryEnabled with AutoMemoryEnabled() and RABBIT_CODE_TEAMMEM (TS: tengu_herring_clock, default false).
func TeamMemoryEnabled() bool {
	return TeamMemoryEnabledFromMerged(nil)
}

// TeamMemoryEnabledFromMerged requires auto memory; then merged["teamMemoryEnabled"] if present, else truthy RABBIT_CODE_TEAMMEM.
func TeamMemoryEnabledFromMerged(merged map[string]interface{}) bool {
	if !AutoMemoryEnabledFromMerged(merged) {
		return false
	}
	if merged != nil {
		if raw, ok := merged["teamMemoryEnabled"]; ok {
			if b, parsed := parseJSONBoolSetting(raw); parsed {
				return b
			}
		}
	}
	return truthy(os.Getenv(EnvTeamMem))
}

// MemorySearchPastContextEnabled gates optional transcript grep guidance in memory system prompts.
func MemorySearchPastContextEnabled() bool {
	return truthy(os.Getenv(EnvMemorySearchPastContext))
}

// ExtractMemoriesEnvEnabled is true when RABBIT_CODE_EXTRACT_MEMORIES is truthy (build flag analogue EXTRACT_MEMORIES).
func ExtractMemoriesEnvEnabled() bool {
	return truthy(os.Getenv(EnvExtractMemories))
}

// ExtractMemoriesAllowed mirrors isExtractModeActive (paths.ts): env on, and interactive or RABBIT_CODE_EXTRACT_MEMORIES_NON_INTERACTIVE.
func ExtractMemoriesAllowed(nonInteractive bool) bool {
	if !ExtractMemoriesEnvEnabled() {
		return false
	}
	if !nonInteractive {
		return true
	}
	return truthy(os.Getenv(EnvExtractMemoriesNonInteractive))
}

// ExtractMemoriesInterval returns N≥1 from RABBIT_CODE_EXTRACT_MEMORIES_INTERVAL (default 1).
func ExtractMemoriesInterval() int {
	s := strings.TrimSpace(os.Getenv(EnvExtractMemoriesInterval))
	if s == "" {
		return 1
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return 1
	}
	return v
}

// ExtractMemoriesSkipIndex mirrors tengu_moth_copse when RABBIT_CODE_EXTRACT_MEMORIES_SKIP_INDEX is truthy.
func ExtractMemoriesSkipIndex() bool {
	return truthy(os.Getenv(EnvExtractMemoriesSkipIndex))
}

// MemorySystemPromptInjectionEnabled is false when RABBIT_CODE_MEMORY_SYSTEM_PROMPT is set and falsy; default true.
func MemorySystemPromptInjectionEnabled() bool {
	v, ok := os.LookupEnv(EnvMemorySystemPrompt)
	if !ok {
		return true
	}
	return truthy(v)
}

// MemoryPromptSkipIndex mirrors tengu_moth_copse for loadMemoryPrompt / combined builders (shared with extract skip index env).
func MemoryPromptSkipIndex() bool {
	return ExtractMemoriesSkipIndex()
}

// KairosDailyLogMemoryEnabled is true when both KAIROS daily-log and active env are truthy.
func KairosDailyLogMemoryEnabled() bool {
	return truthy(os.Getenv(EnvKairosDailyLogMemory)) && truthy(os.Getenv(EnvKairosActive))
}

// CoworkMemoryExtraGuidelineLines returns non-empty lines from RABBIT_CODE_COWORK_MEMORY_EXTRA_GUIDELINES or CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES.
func CoworkMemoryExtraGuidelineLines() []string {
	s := strings.TrimSpace(firstNonEmptyEnvPair(EnvCoworkMemoryExtraGuidelines, "CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES"))
	if s == "" {
		return nil
	}
	return []string{s}
}

// RemoteModeWithoutMemoryDir is true when remote env is on and no remote memory dir is set (auto memory off path).
func RemoteModeWithoutMemoryDir() bool {
	if !truthy(firstNonEmptyEnvPair(EnvRemote, EnvClaudeRemote)) {
		return false
	}
	md := strings.TrimSpace(firstNonEmptyEnvPair(EnvRemoteMemoryDir, EnvClaudeRemoteMemoryDir))
	return md == ""
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

// ContextCollapseSuppressesProactiveAutocompact mirrors shouldAutoCompact when feature('CONTEXT_COLLAPSE') consults
// isContextCollapseEnabled(): suppress proactive autocompact only while collapse is actively managing context.
// Default when RABBIT_CODE_CONTEXT_COLLAPSE is on: suppress (prior rabbit-code behavior). Set RABBIT_CODE_CONTEXT_COLLAPSE_INACTIVE=1
// when the collapse system is disabled at runtime while the env gate remains set.
func ContextCollapseSuppressesProactiveAutocompact() bool {
	if !ContextCollapseEnabled() {
		return false
	}
	if truthy(os.Getenv(EnvContextCollapseInactive)) {
		return false
	}
	return true
}

// CommitAttributionEnabled is the headless analogue of feature('COMMIT_ATTRIBUTION') (postCompactCleanup.ts).
func CommitAttributionEnabled() bool { return truthy(os.Getenv(EnvCommitAttribution)) }

// ExperimentalSkillSearchEnabled mirrors feature('EXPERIMENTAL_SKILL_SEARCH') (compact.ts stripReinjectedAttachments).
func ExperimentalSkillSearchEnabled() bool { return truthy(os.Getenv(EnvExperimentalSkillSearch)) }

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
func SessionRestoreEnabled() bool    { return truthy(os.Getenv(EnvSessionRestore)) }

// TodoWriteToolEnabled mirrors TodoWriteTool.isEnabled (inverse of isTodoV2Enabled in tasks.ts).
func TodoWriteToolEnabled(nonInteractive bool) bool {
	if truthy(os.Getenv(EnvClaudeCodeEnableTasks)) {
		return false
	}
	if nonInteractive {
		return false
	}
	return true
}

// VerificationAgentEnabled mirrors feature('VERIFICATION_AGENT') (default off when env unset).
func VerificationAgentEnabled() bool {
	return truthy(os.Getenv(EnvVerificationAgent))
}

// TenguHiveEvidenceEnabled mirrors getFeatureValue_CACHED_MAY_BE_STALE('tengu_hive_evidence', false).
func TenguHiveEvidenceEnabled() bool {
	return truthy(os.Getenv(EnvTenguHiveEvidence))
}

// TodoWriteVerificationNudgeEnabled is true when both upstream gates are on (TodoWriteTool.ts call() nudge).
func TodoWriteVerificationNudgeEnabled() bool {
	return VerificationAgentEnabled() && TenguHiveEvidenceEnabled()
}

func UltrathinkEnabled() bool        { return truthy(os.Getenv(EnvUltrathink)) }
func UltraplanEnabled() bool         { return truthy(os.Getenv(EnvUltraplan)) }
func BreakCacheCommandEnabled() bool { return truthy(os.Getenv(EnvBreakCacheCommand)) }
func TemplatesEnabled() bool         { return truthy(os.Getenv(EnvTemplates)) }

// StopHooksDir returns RABBIT_CODE_STOP_HOOKS_DIR when set (trimmed); empty if unset.
func StopHooksDir() string            { return strings.TrimSpace(os.Getenv(EnvStopHooksDir)) }
func CachedMicrocompactEnabled() bool { return truthy(os.Getenv(EnvCachedMicrocompact)) }

// CompactStreamingRetryEnabled mirrors getFeatureValue_CACHED_MAY_BE_STALE('tengu_compact_streaming_retry', false).
func CompactStreamingRetryEnabled() bool { return truthy(os.Getenv(EnvCompactStreamingRetry)) }

// CompactCachePrefixEnabled mirrors getFeatureValue_CACHED_MAY_BE_STALE('tengu_compact_cache_prefix', true): when true, try ForkCompactSummary before streaming.
func CompactCachePrefixEnabled() bool {
	s := strings.TrimSpace(os.Getenv(EnvCompactCachePrefix))
	if s == "" {
		return true
	}
	return truthy(s)
}

// CompactStreamingToolSearchEnabled adds ToolSearch to compact stream tools (isToolSearchEnabled analogue; opt-in).
func CompactStreamingToolSearchEnabled() bool { return truthy(os.Getenv(EnvCompactToolSearch)) }

// RemoteSendKeepalivesEnabled mirrors isEnvTruthy(process.env.CLAUDE_CODE_REMOTE_SEND_KEEPALIVES) (optional rabbit-prefixed alias).
func RemoteSendKeepalivesEnabled() bool {
	return truthy(os.Getenv(EnvRemoteSendKeepalives)) || truthy(os.Getenv(EnvRemoteSendKeepalivesRabbit))
}

// SessionMemoryFeatureEnabled mirrors getFeatureValue_CACHED_MAY_BE_STALE('tengu_session_memory', false).
func SessionMemoryFeatureEnabled() bool {
	return truthy(os.Getenv(EnvSessionMemoryFeature))
}

// SessionMemoryCompactFeatureEnabled mirrors getFeatureValue_CACHED_MAY_BE_STALE('tengu_sm_compact', false).
func SessionMemoryCompactFeatureEnabled() bool {
	return truthy(os.Getenv(EnvSessionMemoryCompactFeature))
}

// SessionMemoryCompactionEnabled mirrors shouldUseSessionMemoryCompaction() (env overrides + both flags).
func SessionMemoryCompactionEnabled() bool {
	if truthy(os.Getenv(EnvEnableClaudeCodeSMCompact)) {
		return true
	}
	if truthy(os.Getenv(EnvDisableClaudeCodeSMCompact)) {
		return false
	}
	return SessionMemoryFeatureEnabled() && SessionMemoryCompactFeatureEnabled()
}

// CachedMicrocompactTriggerThreshold default 50 when unset/invalid (headless analogue of cached MC GrowthBook).
func CachedMicrocompactTriggerThreshold() int {
	s := strings.TrimSpace(os.Getenv(EnvCachedMCTriggerThreshold))
	if s == "" {
		return 50
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 50
	}
	return n
}

// CachedMicrocompactKeepRecent default 5 when unset/invalid.
func CachedMicrocompactKeepRecent() int {
	s := strings.TrimSpace(os.Getenv(EnvCachedMCKeepRecent))
	if s == "" {
		return 5
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 5
	}
	return n
}

// SessionMemoryCompactMinTokens mirrors DEFAULT_SM_COMPACT_CONFIG.minTokens (sessionMemoryCompact.ts).
func SessionMemoryCompactMinTokens() int {
	s := strings.TrimSpace(os.Getenv(EnvSMCompactMinTokens))
	if s == "" {
		return 10_000
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 10_000
	}
	return n
}

// SessionMemoryCompactMinTextBlockMessages mirrors DEFAULT_SM_COMPACT_CONFIG.minTextBlockMessages.
func SessionMemoryCompactMinTextBlockMessages() int {
	s := strings.TrimSpace(os.Getenv(EnvSMCompactMinTextMessages))
	if s == "" {
		return 5
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 5
	}
	return n
}

// SessionMemoryCompactMaxTokens mirrors DEFAULT_SM_COMPACT_CONFIG.maxTokens.
func SessionMemoryCompactMaxTokens() int {
	s := strings.TrimSpace(os.Getenv(EnvSMCompactMaxTokens))
	if s == "" {
		return 40_000
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 40_000
	}
	return n
}

// TimeBasedMicrocompactEnabled mirrors timeBasedMCConfig default enabled: false unless env truthy.
func TimeBasedMicrocompactEnabled() bool {
	return truthy(os.Getenv(EnvTimeBasedMicrocompact))
}

// TimeBasedMCGapThresholdMinutes default 60 (safe vs server cache TTL per TS comment).
func TimeBasedMCGapThresholdMinutes() int {
	s := strings.TrimSpace(os.Getenv(EnvTimeBasedMCGapMinutes))
	if s == "" {
		return 60
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 60
	}
	return v
}

// TimeBasedMCKeepRecent default 5 (most recent compactable tool results to keep).
func TimeBasedMCKeepRecent() int {
	s := strings.TrimSpace(os.Getenv(EnvTimeBasedMCKeepRecent))
	if s == "" {
		return 5
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 5
	}
	return v
}

func HistorySnipEnabled() bool { return truthy(os.Getenv(EnvHistorySnip)) }
func SnipCompactEnabled() bool { return truthy(os.Getenv(EnvSnipCompact)) }
func BashExecEnabled() bool    { return truthy(os.Getenv(EnvBashExec)) }

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
