package engine

import (
	"context"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestInstallAnthropicStreamingCompact_setsExecutor(t *testing.T) {
	e := New(context.Background(), &Config{})
	aa := &querydeps.AnthropicAssistant{Client: nil}
	e.InstallAnthropicStreamingCompact(aa, "x")
	if e.compactExecutor == nil {
		t.Fatal("expected compactExecutor")
	}
	_, _, err := e.compactExecutor(context.Background(), compact.RunIdle, []byte(`[{"role":"user","content":[{"type":"text","text":"a"}]}]`))
	if err != querydeps.ErrNilAnthropicClient {
		t.Fatalf("want ErrNilAnthropicClient, got %v", err)
	}
}

func TestInstallAnthropicStreamingCompact_skipsWhenAlreadyConfigured(t *testing.T) {
	var calls int
	e := New(context.Background(), &Config{
		CompactExecutor: func(context.Context, compact.RunPhase, []byte) (string, []byte, error) {
			calls++
			return "", nil, nil
		},
	})
	e.InstallAnthropicStreamingCompact(&querydeps.AnthropicAssistant{}, "")
	_, _, _ = e.compactExecutor(context.Background(), compact.RunIdle, []byte(`[]`))
	if calls != 1 {
		t.Fatalf("custom executor calls=%d", calls)
	}
}
