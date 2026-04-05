package thinking

import (
	"os"
	"testing"
)

func TestHasUltrathinkKeyword(t *testing.T) {
	if HasUltrathinkKeyword("hello") {
		t.Fatal("expected false")
	}
	if !HasUltrathinkKeyword("please ultrathink this") {
		t.Fatal("expected true")
	}
	if !HasUltrathinkKeyword("ULTRATHINK now") {
		t.Fatal("expected true")
	}
}

func TestFindThinkingTriggerPositions(t *testing.T) {
	got := FindThinkingTriggerPositions("a ultrathink and Ultrathink")
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Start != 2 || got[1].Start <= got[0].End {
		t.Fatalf("%+v %+v", got[0], got[1])
	}
}

func TestGetRainbowColor(t *testing.T) {
	if GetRainbowColor(0, false) != "rainbow_red" {
		t.Fatal(GetRainbowColor(0, false))
	}
	if GetRainbowColor(7, false) != "rainbow_red" {
		t.Fatal("mod 7")
	}
	if GetRainbowColor(0, true) != "rainbow_red_shimmer" {
		t.Fatal(GetRainbowColor(0, true))
	}
}

func TestModelSupportsThinking(t *testing.T) {
	if !ModelSupportsThinking("claude-sonnet-4-20250514", ProviderAnthropic) {
		t.Fatal("sonnet 4 1p")
	}
	if ModelSupportsThinking("claude-3-5-haiku-20241022", ProviderAnthropic) {
		t.Fatal("claude 3")
	}
	if !ModelSupportsThinking("anthropic.claude-sonnet-4-20250514-v1:0", ProviderBedrock) {
		t.Fatal("bedrock sonnet 4")
	}
	if ModelSupportsThinking("claude-3-5-sonnet-20241022-v2:0", ProviderBedrock) {
		t.Fatal("bedrock 3")
	}
}

func TestModelSupportsAdaptiveThinking(t *testing.T) {
	if !ModelSupportsAdaptiveThinking("claude-opus-4-6-20250101", ProviderAnthropic) {
		t.Fatal("4-6")
	}
	if ModelSupportsAdaptiveThinking("claude-sonnet-4-20250514", ProviderAnthropic) {
		t.Fatal("4 non-6 sonnet")
	}
	if !ModelSupportsAdaptiveThinking("some-new-model-id", ProviderAnthropic) {
		t.Fatal("unknown 1p defaults true")
	}
	if ModelSupportsAdaptiveThinking("some-new-model-id", ProviderBedrock) {
		t.Fatal("unknown bedrock defaults false")
	}
}

func TestShouldEnableThinkingByDefault(t *testing.T) {
	t.Setenv("MAX_THINKING_TOKENS", "")
	t.Setenv("RABBIT_CODE_MAX_THINKING_TOKENS", "")
	t.Setenv("RABBIT_CODE_ALWAYS_THINKING_DISABLED", "")
	if !ShouldEnableThinkingByDefault() {
		t.Fatal("default true")
	}
	// TS: MAX_THINKING_TOKENS parses to >0 only then short-circuit true; "0" falls through to default (still true).
	t.Setenv("MAX_THINKING_TOKENS", "0")
	if !ShouldEnableThinkingByDefault() {
		t.Fatal("max 0 falls through like TS")
	}
	t.Setenv("MAX_THINKING_TOKENS", "1024")
	if !ShouldEnableThinkingByDefault() {
		t.Fatal("max positive")
	}
	t.Setenv("MAX_THINKING_TOKENS", "")
	t.Setenv("RABBIT_CODE_ALWAYS_THINKING_DISABLED", "1")
	if ShouldEnableThinkingByDefault() {
		t.Fatal("disabled")
	}
	// restore for other tests in process
	_ = os.Unsetenv("RABBIT_CODE_ALWAYS_THINKING_DISABLED")
}
