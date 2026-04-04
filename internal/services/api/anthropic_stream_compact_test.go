package anthropic

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestStartCompactKeepAlive_envOff(t *testing.T) {
	t.Setenv(features.EnvRemoteSendKeepalives, "")
	t.Setenv(features.EnvRemoteSendKeepalivesRabbit, "")
	var n int32
	a := &AnthropicAssistant{
		SessionActivityPing:      func(context.Context) { atomic.AddInt32(&n, 1) },
		CompactKeepAliveInterval: 5 * time.Millisecond,
	}
	stop := a.startCompactKeepAlive(context.Background())
	time.Sleep(25 * time.Millisecond)
	stop()
	if atomic.LoadInt32(&n) != 0 {
		t.Fatalf("expected no pings, got %d", n)
	}
}

func TestStartCompactKeepAlive_envOn(t *testing.T) {
	t.Setenv(features.EnvRemoteSendKeepalives, "1")
	var n int32
	a := &AnthropicAssistant{
		SessionActivityPing:      func(context.Context) { atomic.AddInt32(&n, 1) },
		CompactKeepAliveInterval: 5 * time.Millisecond,
	}
	stop := a.startCompactKeepAlive(context.Background())
	time.Sleep(30 * time.Millisecond)
	stop()
	if atomic.LoadInt32(&n) == 0 {
		t.Fatal("expected at least one ping")
	}
}
