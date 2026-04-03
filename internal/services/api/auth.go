package anthropic

import (
	"context"
	"io"
	"net/http"
	"strings"
)

// AuthTransport sets API key or Bearer OAuth headers (utils/http getAuthHeaders + client.ts).
// Optional Refresh implements withOAuth401Retry: on HTTP 401 with Bearer auth, Refresh runs once
// and the request is retried. Retrying a body-bearing request requires req.GetBody (PostMessagesStream sets it).
type AuthTransport struct {
	Base      http.RoundTripper
	APIKey    string
	Bearer    string
	GetBearer func() string
	Refresh   func(ctx context.Context) error
}

func (t *AuthTransport) bearer() string {
	if t.GetBearer != nil {
		return t.GetBearer()
	}
	return t.Bearer
}

func (t *AuthTransport) usingBearer() bool {
	return strings.TrimSpace(t.APIKey) == "" && strings.TrimSpace(t.bearer()) != ""
}

func (t *AuthTransport) applyAuth(r *http.Request) {
	if strings.TrimSpace(t.APIKey) != "" {
		r.Header.Set("x-api-key", t.APIKey)
		if r.Header.Get("anthropic-version") == "" {
			r.Header.Set("anthropic-version", "2023-06-01")
		}
		return
	}
	b := strings.TrimSpace(t.bearer())
	if b != "" {
		r.Header.Set("Authorization", "Bearer "+b)
	}
}

func (t *AuthTransport) cloneForRoundTrip(req *http.Request) (*http.Request, error) {
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		r := req.Clone(req.Context())
		r.Body = body
		if req.ContentLength >= 0 {
			r.ContentLength = req.ContentLength
		}
		return r, nil
	}
	return req.Clone(req.Context()), nil
}

// RoundTrip implements http.RoundTripper.
func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTrip(req, false)
}

func (t *AuthTransport) roundTrip(req *http.Request, after401 bool) (*http.Response, error) {
	r, err := t.cloneForRoundTrip(req)
	if err != nil {
		return nil, err
	}
	t.applyAuth(r)
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized || after401 || t.Refresh == nil || !t.usingBearer() {
		return resp, nil
	}
	if req.Body != nil && req.GetBody == nil {
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if err := t.Refresh(req.Context()); err != nil {
		return nil, err
	}
	return t.roundTrip(req, true)
}
