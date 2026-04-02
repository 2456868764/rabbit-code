package app

import (
	"context"

	"github.com/2456868764/rabbit-code/internal/anthropic"
)

// NewAnthropicClient builds a Messages API client using the same outbound resolution as Bootstrap
// (ResolveAPIOutboundTransport), API key from ReadAPIKey, X-Claude-Code-Session-Id from rt.State,
// and stream usage wired to bootstrap state (P4.4.1). Returns nil if rt is nil.
func NewAnthropicClient(ctx context.Context, rt *Runtime) *anthropic.Client {
	if rt == nil {
		return nil
	}
	cl := anthropic.NewClientWithPool(ctx, rt.RootCAs, ReadAPIKey(rt.GlobalConfigDir), "")
	cl.SetOnStreamUsageBootstrap(rt.State)
	if rt.State != nil {
		cl.SessionID = rt.State.SessionID()
	}
	return cl
}
