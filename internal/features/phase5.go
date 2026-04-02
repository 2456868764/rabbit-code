package features

import (
	"os"
	"strconv"
	"strings"
)

// Phase 5 runtime env mirrors SOURCE_FEATURE_FLAGS.md (TOKEN_BUDGET, REACTIVE_COMPACT, …).
// Defaults are off (unset/falsy). Full upstream behavior is deferred; see docs/phases/PARITY_PHASE5_DEFERRED.md.
const (
	EnvTokenBudget            = "RABBIT_CODE_TOKEN_BUDGET"
	EnvTokenBudgetMaxInputBytes = "RABBIT_CODE_TOKEN_BUDGET_MAX_INPUT_BYTES"
	EnvReactiveCompact        = "RABBIT_CODE_REACTIVE_COMPACT"
	EnvContextCollapse        = "RABBIT_CODE_CONTEXT_COLLAPSE"
	EnvUltrathink             = "RABBIT_CODE_ULTRATHINK"
	EnvUltraplan              = "RABBIT_CODE_ULTRAPLAN"
	EnvBreakCacheCommand      = "RABBIT_CODE_BREAK_CACHE_COMMAND"
	EnvTemplates              = "RABBIT_CODE_TEMPLATES"
	EnvCachedMicrocompact     = "RABBIT_CODE_CACHED_MICROCOMPACT"
	EnvHistorySnip            = "RABBIT_CODE_HISTORY_SNIP"
	// EnvSnipCompact gates snip-style transcript trimming hooks (pairs with query.SnipDropFirstMessages, P5.2.2).
	EnvSnipCompact = "RABBIT_CODE_SNIP_COMPACT"
)

func TokenBudgetEnabled() bool { return truthy(os.Getenv(EnvTokenBudget)) }

// TokenBudgetMaxInputBytes enforces a UTF-8 byte cap on resolved Submit text when TOKEN_BUDGET is on.
// Returns 0 when the budget feature is off (no enforcement). When on and env unset, defaults to 4_000_000.
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

func ReactiveCompactEnabled() bool   { return truthy(os.Getenv(EnvReactiveCompact)) }
func ContextCollapseEnabled() bool    { return truthy(os.Getenv(EnvContextCollapse)) }
func UltrathinkEnabled() bool         { return truthy(os.Getenv(EnvUltrathink)) }
func UltraplanEnabled() bool          { return truthy(os.Getenv(EnvUltraplan)) }
func BreakCacheCommandEnabled() bool  { return truthy(os.Getenv(EnvBreakCacheCommand)) }
func TemplatesEnabled() bool          { return truthy(os.Getenv(EnvTemplates)) }
func CachedMicrocompactEnabled() bool { return truthy(os.Getenv(EnvCachedMicrocompact)) }
func HistorySnipEnabled() bool        { return truthy(os.Getenv(EnvHistorySnip)) }
func SnipCompactEnabled() bool        { return truthy(os.Getenv(EnvSnipCompact)) }

// PromptCacheBreakDetectionEnabled aliases Phase 4 env (P5.F.9 shares anthropic client gates).
func PromptCacheBreakDetectionEnabled() bool { return PromptCacheBreakDetection() }
