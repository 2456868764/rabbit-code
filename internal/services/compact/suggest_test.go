package compact

import (
	"context"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
)

func TestReactiveSuggestFromTranscript(t *testing.T) {
	if !ReactiveSuggestFromTranscript([]byte("abcde"), 0, 2) {
		t.Fatal()
	}
}

func TestTranscriptReactiveSuggest_respectsLoopState(t *testing.T) {
	st := &query.LoopState{HasAttemptedReactiveCompact: true}
	if TranscriptReactiveSuggest(st, []byte("abcde"), 0, 2) {
		t.Fatal("expected suppressed when HasAttemptedReactiveCompact")
	}
	st.HasAttemptedReactiveCompact = false
	if !TranscriptReactiveSuggest(st, []byte("abcde"), 0, 2) {
		t.Fatal("expected suggest when flag cleared")
	}
}

func TestExecuteStubWithMeta(t *testing.T) {
	s, next, err := ExecuteStubWithMeta(context.Background(), RunIdle, []byte(`{"a":1}`))
	if len(next) != 0 {
		t.Fatalf("unexpected next transcript: %q", next)
	}
	if err != nil || !strings.Contains(s, "estTok=") || !strings.Contains(s, "bytes=") {
		t.Fatalf("%q %v", s, err)
	}
}
