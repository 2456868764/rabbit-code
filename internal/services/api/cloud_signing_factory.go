package anthropic

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
)

// NewAPIOutboundTransport builds HTTPTransportAPIOutboundWithRoots(pool) then applies NewSigningTransportForProvider
// for the current DetectProvider() (Bedrock SigV4, Vertex Bearer; 1P/Foundry unchanged). Use from Bootstrap / CLI wiring (AC4-6).
func NewAPIOutboundTransport(ctx context.Context, pool *x509.CertPool) (http.RoundTripper, error) {
	base, err := HTTPTransportAPIOutboundWithRoots(pool)
	if err != nil {
		return nil, err
	}
	return NewSigningTransportForProvider(ctx, base, DetectProvider())
}

// ResolveAPIOutboundTransport tries NewAPIOutboundTransport; on error returns HTTPTransportWithProxyFromEnvAndRoots(pool)
// and the original error so callers can log (Bootstrap). Messages client factories use the same resolution.
func ResolveAPIOutboundTransport(ctx context.Context, pool *x509.CertPool) (rt http.RoundTripper, err error) {
	t, e := NewAPIOutboundTransport(ctx, pool)
	if e == nil {
		return t, nil
	}
	return HTTPTransportWithProxyFromEnvAndRoots(pool), e
}

// NewSigningTransportForProvider wraps base with SigV4 (Bedrock) or GCP Bearer (Vertex) when applicable.
// For ProviderAnthropic and ProviderFoundry, returns base unchanged (Foundry Azure signing not implemented yet).
func NewSigningTransportForProvider(ctx context.Context, base http.RoundTripper, p Provider) (http.RoundTripper, error) {
	if base == nil {
		base = http.DefaultTransport
	}
	switch p {
	case ProviderBedrock:
		s, err := NewBedrockSigV4Signer(ctx)
		if err != nil {
			return nil, err
		}
		return &SigningTransport{Base: base, Signer: s}, nil
	case ProviderVertex:
		s, err := NewVertexTokenSigner(ctx)
		if err != nil {
			return nil, err
		}
		return &SigningTransport{Base: base, Signer: s}, nil
	default:
		return base, nil
	}
}

// MustNewSigningTransportForProvider is like NewSigningTransportForProvider but panics on error (startup-only helpers).
func MustNewSigningTransportForProvider(ctx context.Context, base http.RoundTripper, p Provider) http.RoundTripper {
	t, err := NewSigningTransportForProvider(ctx, base, p)
	if err != nil {
		panic(fmt.Errorf("anthropic: NewSigningTransportForProvider: %w", err))
	}
	return t
}
