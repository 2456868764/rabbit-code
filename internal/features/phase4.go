package features

import (
	"os"
	"strings"
)

// Phase 4 env mirrors SOURCE_FEATURE_FLAGS / client.ts / withRetry.ts (rabbit-code prefixed).
const (
	EnvUnattendedRetry       = "RABBIT_CODE_UNATTENDED_RETRY"
	// EnvFastRetry shortens HTTP retry backoff (withRetry.ts fast path); mirrors CLAUDE_CODE_FAST_RETRY when set.
	EnvFastRetry = "RABBIT_CODE_FAST_RETRY"
	EnvAdditionalProtection  = "RABBIT_CODE_ADDITIONAL_PROTECTION"
	EnvUseBedrock            = "RABBIT_CODE_USE_BEDROCK"
	EnvUseVertex             = "RABBIT_CODE_USE_VERTEX"
	EnvUseFoundry            = "RABBIT_CODE_USE_FOUNDRY"
	EnvSkipBedrockAuth       = "RABBIT_CODE_SKIP_BEDROCK_AUTH"
	EnvSkipVertexAuth        = "RABBIT_CODE_SKIP_VERTEX_AUTH"
	EnvSkipFoundryAuth       = "RABBIT_CODE_SKIP_FOUNDRY_AUTH"
	EnvAntiDistillation      = "RABBIT_CODE_ANTI_DISTILLATION_CC"
	// EnvAntiDistillationHeader overrides the HTTP header name when anti-distillation is on (default x-rabbit-code-anti-distillation).
	EnvAntiDistillationHeader = "RABBIT_CODE_ANTI_DISTILLATION_HEADER"
	// EnvAntiDistillationValue sets the header value (default 1).
	EnvAntiDistillationValue = "RABBIT_CODE_ANTI_DISTILLATION_VALUE"
	// EnvOAuthBetaAppend is a comma-separated list of extra anthropic-beta names for OAuth sessions (constants/oauth.ts OAUTH_BETA_HEADER patterns).
	EnvOAuthBetaAppend = "RABBIT_CODE_OAUTH_BETA_APPEND"
	EnvNativeAttestation     = "RABBIT_CODE_NATIVE_CLIENT_ATTESTATION"
	// EnvNativeAttestationHeader overrides the HTTP header name when native attestation is on.
	EnvNativeAttestationHeader = "RABBIT_CODE_NATIVE_ATTESTATION_HEADER"
	// EnvNativeAttestationValue sets the header value (default 1).
	EnvNativeAttestationValue = "RABBIT_CODE_NATIVE_ATTESTATION_VALUE"
	EnvPromptCacheBreak      = "RABBIT_CODE_PROMPT_CACHE_BREAK_DETECTION"
	EnvE2EMockAPI            = "RABBIT_CODE_E2E_MOCK_API"
	// EnvOAuthBaseURL overrides console base for usage.ts fetchUtilization (default https://console.anthropic.com).
	EnvOAuthBaseURL = "RABBIT_CODE_OAUTH_BASE_URL"
	// EnvHTTPUserAgent overrides default User-Agent for API HTTP clients (utils/http.ts).
	EnvHTTPUserAgent = "RABBIT_CODE_USER_AGENT"
	// EnvStrictForeground529 sets DefaultPolicy().StrictForeground529 (withRetry.ts FOREGROUND_529_RETRY_SOURCES gate for HTTP 529).
	EnvStrictForeground529 = "RABBIT_CODE_STRICT_FOREGROUND_529"
	// EnvAttributionHeader when set to a falsy value disables the billing attribution system line (CLAUDE_CODE_ATTRIBUTION_HEADER); unset = enabled.
	EnvAttributionHeader = "RABBIT_CODE_ATTRIBUTION_HEADER"
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

func UseBedrock() bool    { return truthy(os.Getenv(EnvUseBedrock)) }
func UseVertex() bool     { return truthy(os.Getenv(EnvUseVertex)) }
func UseFoundry() bool    { return truthy(os.Getenv(EnvUseFoundry)) }
func SkipBedrockAuth() bool { return truthy(os.Getenv(EnvSkipBedrockAuth)) }
func SkipVertexAuth() bool  { return truthy(os.Getenv(EnvSkipVertexAuth)) }
func SkipFoundryAuth() bool { return truthy(os.Getenv(EnvSkipFoundryAuth)) }

func AntiDistillationCC() bool {
	return truthy(os.Getenv(EnvAntiDistillation))
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

// AttributionHeaderPromptEnabled mirrors system.ts isAttributionHeaderEnabled: default true unless RABBIT_CODE_ATTRIBUTION_HEADER is set and falsy.
func AttributionHeaderPromptEnabled() bool {
	v, ok := os.LookupEnv(EnvAttributionHeader)
	if !ok {
		return true
	}
	return truthy(v)
}
