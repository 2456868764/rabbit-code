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
	// EnvSnipCompactMaxBytes / EnvSnipCompactMaxRounds: when SNIP_COMPACT is on, trim prefix each assistant round (independent of HISTORY_SNIP).
	EnvSnipCompactMaxBytes  = "RABBIT_CODE_SNIP_COMPACT_MAX_BYTES"
	EnvSnipCompactMaxRounds = "RABBIT_CODE_SNIP_COMPACT_MAX_ROUNDS"
	// EnvReactiveCompactMinBytes: when REACTIVE_COMPACT is on, suggest reactive compact if len(transcript JSON) >= this (default 8192).
	EnvReactiveCompactMinBytes = "RABBIT_CODE_REACTIVE_COMPACT_MIN_BYTES"
	// EnvHistorySnipMaxBytes / EnvHistorySnipMaxRounds gate P5.F.10 transcript prefix drops each assistant round.
	EnvHistorySnipMaxBytes  = "RABBIT_CODE_HISTORY_SNIP_MAX_BYTES"
	EnvHistorySnipMaxRounds = "RABBIT_CODE_HISTORY_SNIP_MAX_ROUNDS"
	// EnvTemplateNames comma-separated names emitted with EventKindTemplatesActive when TEMPLATES is on.
	EnvTemplateNames = "RABBIT_CODE_TEMPLATE_NAMES"
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

// SnipCompactMaxBytes returns 0 when SNIP_COMPACT is off.
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

// SnipCompactMaxRounds returns max prefix drops per assistant iteration when SNIP_COMPACT is on.
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

// PromptCacheBreakDetectionEnabled aliases Phase 4 env (P5.F.9 shares anthropic client gates).
func PromptCacheBreakDetectionEnabled() bool { return PromptCacheBreakDetection() }

// ReactiveCompactMinTranscriptBytes returns 0 when REACTIVE_COMPACT is off; else min JSON byte length to force reactive suggest.
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

// HistorySnipMaxBytes returns 0 when HISTORY_SNIP is off or unset invalid.
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

// HistorySnipMaxRounds returns max SnipDropFirstMessages calls per assistant iteration when HISTORY_SNIP is on.
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

// TemplateNames returns comma-separated template ids from RABBIT_CODE_TEMPLATE_NAMES when TEMPLATES is enabled.
func TemplateNames() []string {
	if !TemplatesEnabled() {
		return nil
	}
	return splitCommaEnv(os.Getenv(EnvTemplateNames))
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
