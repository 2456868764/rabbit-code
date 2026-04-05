package thinking

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Provider mirrors anthropic.Provider iota order (internal/services/api/provider.go) for thinking gates only.
type Provider uint8

const (
	ProviderAnthropic Provider = iota
	ProviderBedrock
	ProviderVertex
	ProviderFoundry
)

// Word-boundary ultrathink (thinking.ts hasUltrathinkKeyword / findThinkingTriggerPositions).
var ultrathinkWordRe = regexp.MustCompile(`(?i)\bultrathink\b`)

// IsUltrathinkEnabled mirrors isUltrathinkEnabled: build/feature gate + rollout.
// Go: RABBIT_CODE_ULTRATHINK (features.UltrathinkEnabled); TS also requires ULTRATHINK bundle + GrowthBook.
func IsUltrathinkEnabled() bool {
	return features.UltrathinkEnabled()
}

// HasUltrathinkKeyword mirrors hasUltrathinkKeyword.
func HasUltrathinkKeyword(text string) bool {
	return ultrathinkWordRe.MatchString(text)
}

// ThinkingTriggerPosition is one match for UI highlight (findThinkingTriggerPositions).
type ThinkingTriggerPosition struct {
	Word  string
	Start int
	End   int
}

// FindThinkingTriggerPositions mirrors findThinkingTriggerPositions (fresh /g each call).
func FindThinkingTriggerPositions(text string) []ThinkingTriggerPosition {
	var out []ThinkingTriggerPosition
	for _, ix := range ultrathinkWordRe.FindAllStringIndex(text, -1) {
		if len(ix) == 2 {
			out = append(out, ThinkingTriggerPosition{
				Word:  text[ix[0]:ix[1]],
				Start: ix[0],
				End:   ix[1],
			})
		}
	}
	return out
}

// Rainbow color names for lipgloss / TUI (Theme keys in TS).
var (
	rainbowColors        = []string{"rainbow_red", "rainbow_orange", "rainbow_yellow", "rainbow_green", "rainbow_blue", "rainbow_indigo", "rainbow_violet"}
	rainbowShimmerColors = []string{"rainbow_red_shimmer", "rainbow_orange_shimmer", "rainbow_yellow_shimmer", "rainbow_green_shimmer", "rainbow_blue_shimmer", "rainbow_indigo_shimmer", "rainbow_violet_shimmer"}
)

// GetRainbowColor mirrors getRainbowColor(charIndex, shimmer).
func GetRainbowColor(charIndex int, shimmer bool) string {
	colors := rainbowColors
	if shimmer {
		colors = rainbowShimmerColors
	}
	if len(colors) == 0 {
		return ""
	}
	if charIndex < 0 {
		charIndex = -charIndex
	}
	return colors[charIndex%len(colors)]
}

func canonicalModelLower(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}

// ModelSupportsThinking mirrors modelSupportsThinking (no get3PModelCapabilityOverride / ant allowlist yet).
func ModelSupportsThinking(model string, p Provider) bool {
	c := canonicalModelLower(model)
	switch p {
	case ProviderAnthropic, ProviderFoundry:
		return !strings.Contains(c, "claude-3-")
	case ProviderBedrock, ProviderVertex:
		return strings.Contains(c, "sonnet-4") || strings.Contains(c, "opus-4")
	default:
		return strings.Contains(c, "sonnet-4") || strings.Contains(c, "opus-4")
	}
}

// ModelSupportsAdaptiveThinking mirrors modelSupportsAdaptiveThinking (no 3P override).
func ModelSupportsAdaptiveThinking(model string, p Provider) bool {
	c := canonicalModelLower(model)
	if strings.Contains(c, "opus-4-6") || strings.Contains(c, "sonnet-4-6") {
		return true
	}
	if strings.Contains(c, "opus") || strings.Contains(c, "sonnet") || strings.Contains(c, "haiku") {
		return false
	}
	return p == ProviderAnthropic || p == ProviderFoundry
}

// ShouldEnableThinkingByDefault mirrors shouldEnableThinkingByDefault (no merged settings alwaysThinkingEnabled yet).
func ShouldEnableThinkingByDefault() bool {
	if v := strings.TrimSpace(os.Getenv("MAX_THINKING_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return true
		}
	}
	if v := strings.TrimSpace(os.Getenv("RABBIT_CODE_MAX_THINKING_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return true
		}
	}
	// TS: settings.alwaysThinkingEnabled === false → false. Go: RABBIT_CODE_ALWAYS_THINKING_DISABLED=1.
	if truthyEnv("RABBIT_CODE_ALWAYS_THINKING_DISABLED") {
		return false
	}
	return true
}

func truthyEnv(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
