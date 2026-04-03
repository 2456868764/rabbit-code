package query

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestBuildHeadlessContextReport_thresholdAndWarnings(t *testing.T) {
	blob := []byte(strings.Repeat("a", 10_000))
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	r := BuildHeadlessContextReport(blob, "m", 1024, 0, 0, "")
	if r.AutoCompactThreshold <= 0 {
		t.Fatalf("threshold %d", r.AutoCompactThreshold)
	}
	if r.EstimatedTokens <= 0 {
		t.Fatal("est tokens")
	}
	if r.ProactiveAutoCompactBlocked {
		t.Fatal("expected not blocked")
	}
}

func TestBuildHeadlessContextReport_sessionMemoryBlocked(t *testing.T) {
	blob := []byte(`[]`)
	t.Setenv(features.EnvContextWindowTokens, "50000")
	r := BuildHeadlessContextReport(blob, "m", 1024, 0, 0, QuerySourceSessionMemory)
	if !r.ProactiveAutoCompactBlocked {
		t.Fatal("expected blocked for session_memory source")
	}
}
