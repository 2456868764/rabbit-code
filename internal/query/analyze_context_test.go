package query

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestReactiveCompactByTranscript_bytesOnly(t *testing.T) {
	if ReactiveCompactByTranscript([]byte("hi"), 1, 0) != true {
		t.Fatal()
	}
	if ReactiveCompactByTranscript([]byte(""), 1, 0) != false {
		t.Fatal()
	}
}

func TestReactiveCompactByTranscript_tokensOnly(t *testing.T) {
	if ReactiveCompactByTranscript([]byte("abcd"), 0, 2) != false {
		t.Fatal()
	}
	if ReactiveCompactByTranscript([]byte("abcde"), 0, 2) != true {
		t.Fatal()
	}
}

func TestReactiveCompactByTranscript_disabledWhenDisableCompact(t *testing.T) {
	t.Setenv(features.EnvDisableCompact, "1")
	if ReactiveCompactByTranscript([]byte("long"), 1, 0) {
		t.Fatal("expected false when RABBIT_CODE_DISABLE_COMPACT")
	}
}

func TestTranscriptReactiveCompactSuggested_respectsHasAttemptedFlag(t *testing.T) {
	st := &LoopState{HasAttemptedReactiveCompact: true}
	if TranscriptReactiveCompactSuggested(st, []byte("long-enough-bytes"), 1, 0) {
		t.Fatal("expected suppressed when HasAttemptedReactiveCompact")
	}
	st.HasAttemptedReactiveCompact = false
	if !TranscriptReactiveCompactSuggested(st, []byte("long-enough-bytes"), 1, 0) {
		t.Fatal("expected true when flag cleared")
	}
}

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

func TestBuildHeadlessContextReport_contextCollapseInactiveUnblocksProactive(t *testing.T) {
	blob := []byte(strings.Repeat("a", 10_000))
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "1")
	t.Setenv(features.EnvContextCollapseInactive, "1")
	r := BuildHeadlessContextReport(blob, "m", 1024, 0, 0, "")
	if r.ProactiveAutoCompactBlocked {
		t.Fatal("expected proactive not blocked when CONTEXT_COLLAPSE_INACTIVE=1")
	}
}

func TestProactiveAutoCompactAllowedForQuerySource(t *testing.T) {
	if !ProactiveAutoCompactAllowedForQuerySource("") {
		t.Fatal("main loop empty source should allow")
	}
	if !ProactiveAutoCompactAllowedForQuerySource("main") {
		t.Fatal("arbitrary non-fork source should allow")
	}
	if ProactiveAutoCompactAllowedForQuerySource(QuerySourceSessionMemory) {
		t.Fatal("session_memory should block")
	}
	if ProactiveAutoCompactAllowedForQuerySource(QuerySourceCompact) {
		t.Fatal("compact should block")
	}
	if ProactiveAutoCompactAllowedForQuerySource(QuerySourceExtractMemories) {
		t.Fatal("extract_memories should block")
	}
	t.Run("marble_origami blocked when context collapse on", func(t *testing.T) {
		t.Setenv(features.EnvContextCollapse, "1")
		if ProactiveAutoCompactAllowedForQuerySource(QuerySourceMarbleOrigami) {
			t.Fatal("expected block")
		}
	})
	t.Run("marble_origami allowed when collapse off", func(t *testing.T) {
		t.Setenv(features.EnvContextCollapse, "")
		if !ProactiveAutoCompactAllowedForQuerySource(QuerySourceMarbleOrigami) {
			t.Fatal("expected allow")
		}
	})
}
