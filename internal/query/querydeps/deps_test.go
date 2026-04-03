package querydeps

import (
	"context"
	"errors"
	"testing"
)

func TestNoopToolRunner(t *testing.T) {
	var r NoopToolRunner
	_, err := r.RunTool(context.Background(), "x", nil)
	if !errors.Is(err, ErrNoToolRunner) {
		t.Fatalf("got %v", err)
	}
}

func TestNoopStreamAssistant(t *testing.T) {
	var s NoopStreamAssistant
	out, err := s.StreamAssistant(context.Background(), "m", 1, []byte(`[]`))
	if err != nil || out != "" {
		t.Fatalf("%q %v", out, err)
	}
}
