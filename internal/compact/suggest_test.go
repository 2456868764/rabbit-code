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
	s, err := ExecuteStubWithMeta(context.Background(), RunIdle, []byte(`{"a":1}`))
	if err != nil || !strings.Contains(s, "estTok=") || !strings.Contains(s, "bytes=") {
		t.Fatalf("%q %v", s, err)
	}
}
