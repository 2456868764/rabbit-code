package query

import "testing"

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
