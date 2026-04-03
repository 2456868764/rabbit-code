package compact

import (
	"context"
	"strings"
	"testing"
)

func TestReactiveSuggestFromTranscript(t *testing.T) {
	if !ReactiveSuggestFromTranscript([]byte("abcde"), 0, 2) {
		t.Fatal()
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
