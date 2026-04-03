package query

import (
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
	// "abcd" -> 1 token by heuristic; minTokens 2 -> false
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
