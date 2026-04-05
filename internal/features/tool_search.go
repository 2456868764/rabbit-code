package features

import (
	"os"
	"strconv"
	"strings"
)

// Upstream env names (toolSearch.ts) — not RABBIT_CODE-prefixed.
const (
	EnvEnableToolSearch             = "ENABLE_TOOL_SEARCH"
	EnvDisableExperimentalBetas     = "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"
	EnvAnthropicBaseURL             = "ANTHROPIC_BASE_URL"
	EnvToolSearchForceOptimistic    = "RABBIT_CODE_TOOL_SEARCH_OPTIMISTIC"
)

// ToolSearchMode mirrors utils/toolSearch.ts ToolSearchMode.
type ToolSearchMode string

const (
	ToolSearchModeTST      ToolSearchMode = "tst"
	ToolSearchModeTSTAuto  ToolSearchMode = "tst-auto"
	ToolSearchModeStandard ToolSearchMode = "standard"
)

func envDefinedFalsyString(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "0" || s == "false" || s == "no" || s == "off"
}

func isAutoToolSearchMode(value string) bool {
	v := strings.TrimSpace(value)
	return v == "auto" || strings.HasPrefix(v, "auto:")
}

// parseToolSearchAutoPercent mirrors parseAutoPercentage in toolSearch.ts; ok false if not auto:N or invalid.
func parseToolSearchAutoPercent(value string) (percent int, ok bool) {
	v := strings.TrimSpace(value)
	if !strings.HasPrefix(v, "auto:") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(v[5:]))
	if err != nil {
		return 0, false
	}
	if n < 0 {
		n = 0
	}
	if n > 100 {
		n = 100
	}
	return n, true
}

// GetToolSearchMode mirrors getToolSearchMode() in utils/toolSearch.ts.
func GetToolSearchMode() ToolSearchMode {
	if truthy(os.Getenv(EnvDisableExperimentalBetas)) {
		return ToolSearchModeStandard
	}
	value, set := os.LookupEnv(EnvEnableToolSearch)
	if !set {
		return ToolSearchModeTST
	}
	v := strings.TrimSpace(value)
	if v == "" {
		return ToolSearchModeTST
	}
	if p, ok := parseToolSearchAutoPercent(v); ok {
		if p == 0 {
			return ToolSearchModeTST
		}
		if p == 100 {
			return ToolSearchModeStandard
		}
		return ToolSearchModeTSTAuto
	}
	if isAutoToolSearchMode(v) {
		return ToolSearchModeTSTAuto
	}
	if truthy(v) {
		return ToolSearchModeTST
	}
	if envDefinedFalsyString(v) {
		return ToolSearchModeStandard
	}
	return ToolSearchModeTST
}

// toolSearchAnthropicURLAllowsOptimistic mirrors isFirstPartyAnthropicBaseUrl heuristic for headless.
func toolSearchAnthropicURLAllowsOptimistic() bool {
	base := strings.TrimSpace(os.Getenv(EnvAnthropicBaseURL))
	if base == "" {
		return true
	}
	u := strings.ToLower(base)
	return strings.Contains(u, "api.anthropic.com") || strings.Contains(u, "console.anthropic.com")
}

// ToolSearchEnabledOptimistic mirrors isToolSearchEnabledOptimistic() in utils/toolSearch.ts.
func ToolSearchEnabledOptimistic() bool {
	if truthy(os.Getenv(EnvToolSearchForceOptimistic)) {
		return true
	}
	if GetToolSearchMode() == ToolSearchModeStandard {
		return false
	}
	_, set := os.LookupEnv(EnvEnableToolSearch)
	if !set || strings.TrimSpace(os.Getenv(EnvEnableToolSearch)) == "" {
		if !toolSearchAnthropicURLAllowsOptimistic() {
			return false
		}
	}
	return true
}
