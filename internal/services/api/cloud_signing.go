package anthropic

import (
	"context"
	"errors"
	"net/http"
)

// ErrCloudSigningNotImplemented is returned by placeholder signers until real SigV4 / GCP wiring lands (AC4-6).
var ErrCloudSigningNotImplemented = errors.New("anthropic: cloud request signing not implemented")

// CloudRequestSigner mutates req before it is sent (AWS SigV4, GCP access tokens, etc.).
// Implementations must not read or consume req.Body unless they restore it so the base RoundTripper can send the body.
type CloudRequestSigner interface {
	Sign(ctx context.Context, req *http.Request) error
}

// SigningTransport runs Signer.Sign before Base.RoundTrip. When Signer is nil, behaves like Base alone.
type SigningTransport struct {
	Base   http.RoundTripper
	Signer CloudRequestSigner
}

// RoundTrip implements http.RoundTripper.
func (t *SigningTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Signer != nil {
		if err := t.Signer.Sign(req.Context(), req); err != nil {
			return nil, err
		}
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// StubBedrockSigner returns ErrCloudSigningNotImplemented from Sign (documents the hook for Bedrock Runtime).
type StubBedrockSigner struct{}

func (StubBedrockSigner) Sign(context.Context, *http.Request) error {
	return ErrCloudSigningNotImplemented
}

// StubVertexSigner returns ErrCloudSigningNotImplemented from Sign (Vertex / GCP access token hook; AC4-6).
type StubVertexSigner struct{}

func (StubVertexSigner) Sign(context.Context, *http.Request) error {
	return ErrCloudSigningNotImplemented
}

// StubFoundrySigner returns ErrCloudSigningNotImplemented from Sign (Azure Foundry / managed identity hook; AC4-6).
type StubFoundrySigner struct{}

func (StubFoundrySigner) Sign(context.Context, *http.Request) error {
	return ErrCloudSigningNotImplemented
}
