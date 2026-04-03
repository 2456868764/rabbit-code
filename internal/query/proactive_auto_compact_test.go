package query

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestProactiveAutoCompactSuggested_gates(t *testing.T) {
	// ~150k UTF-8 bytes → heuristic tokens ~37.5k; with 50k context window threshold ~35.9k → fires.
	blob := []byte(strings.Repeat("a", 150_000))
	t.Run("disabled compact", func(t *testing.T) {
		t.Setenv(features.EnvDisableCompact, "1")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false")
		}
	})
	t.Run("context collapse suppresses proactive", func(t *testing.T) {
		t.Setenv(features.EnvDisableCompact, "")
		t.Setenv(features.EnvContextCollapse, "1")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false")
		}
	})
	t.Run("suppress proactive", func(t *testing.T) {
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvSuppressProactiveAutoCompact, "1")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false")
		}
	})
	t.Run("above threshold", func(t *testing.T) {
		t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
		t.Setenv(features.EnvDisableCompact, "")
		t.Setenv(features.EnvDisableAutoCompact, "")
		t.Setenv(features.EnvAutoCompact, "")
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if !ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected true when transcript exceeds autocompact threshold")
		}
	})
	t.Run("querySource session_memory blocks", func(t *testing.T) {
		t.Setenv(features.EnvDisableCompact, "")
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggestedWithSource(blob, "m", 1024, 0, 0, QuerySourceSessionMemory) {
			t.Fatal("expected false for forked session_memory")
		}
	})
	t.Run("reactive compact plus cobalt suppresses proactive", func(t *testing.T) {
		t.Setenv(features.EnvReactiveCompact, "1")
		t.Setenv(features.EnvTenguCobaltRaccoon, "1")
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false when reactive+cobalt")
		}
	})
}
