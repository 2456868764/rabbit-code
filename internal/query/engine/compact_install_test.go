package engine

import (
	"context"
	"testing"

	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestInstallAnthropicStreamingCompact_setsExecutor(t *testing.T) {
	e := New(context.Background(), &Config{})
	aa := &anthropic.AnthropicAssistant{Client: nil}
	e.InstallAnthropicStreamingCompact(aa, "x")
	if e.compactExecutor == nil {
		t.Fatal("expected compactExecutor")
	}
	_, _, err := e.compactExecutor(context.Background(), compact.RunIdle, []byte(`[{"role":"user","content":[{"type":"text","text":"a"}]}]`))
	if err != anthropic.ErrNilAnthropicClient {
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
	e.InstallAnthropicStreamingCompact(&anthropic.AnthropicAssistant{}, "")
	_, _, _ = e.compactExecutor(context.Background(), compact.RunIdle, []byte(`[]`))
	if calls != 1 {
		t.Fatalf("custom executor calls=%d", calls)
	}
}
