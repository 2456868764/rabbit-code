package anthropic

import (
	"context"
	"crypto/x509"
)

// NewClientWithPool builds a Client whose transport is AuthTransport → ResolveAPIOutboundTransport (proxy + mTLS + cloud signing).
// apiKey and bearer follow NewTransportChain semantics (API key wins when non-empty).
func NewClientWithPool(ctx context.Context, pool *x509.CertPool, apiKey, bearer string) *Client {
	rt, _ := ResolveAPIOutboundTransport(ctx, pool)
	return NewClient(NewTransportChain(rt, apiKey, bearer))
}

// NewClientWithPoolOAuth is like NewClientWithPool but uses dynamic Bearer + optional 401 Refresh (withOAuth401Retry).
func NewClientWithPoolOAuth(ctx context.Context, pool *x509.CertPool, getBearer func() string, refresh func(context.Context) error) *Client {
	rt, _ := ResolveAPIOutboundTransport(ctx, pool)
	return NewClient(NewTransportChainOAuth(rt, getBearer, refresh))
}
