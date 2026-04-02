package services

import (
	"context"
	"net/http"

	"github.com/2456868764/rabbit-code/internal/anthropic"
)

// RoundTripProbe builds a probe request for the named services/api TS file and executes it with
// anthropic.DoRequest (same retry/transient semantics as Messages). Use for P4.6.1 smoke tests
// against mock bases.
func RoundTripProbe(ctx context.Context, rt http.RoundTripper, tsFile, anthropicBase, oauthBase string, pol anthropic.Policy) (*http.Response, error) {
	if rt == nil {
		rt = http.DefaultTransport
	}
	req, err := BuildRequest(tsFile, anthropicBase, oauthBase)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	return anthropic.DoRequest(ctx, rt, req, pol)
}
