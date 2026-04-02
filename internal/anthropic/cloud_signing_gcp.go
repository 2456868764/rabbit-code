package anthropic

import (
	"context"
	"fmt"
	"net/http"

	"github.com/2456868764/rabbit-code/internal/features"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// VertexTokenSigner adds a GCP OAuth2 access token (Application Default Credentials) for Vertex AI (AC4-6).
type VertexTokenSigner struct {
	ts   oauth2.TokenSource
	skip bool
}

// vertexOAuthScope is the scope used for Vertex / Cloud ML API calls with user or service credentials.
const vertexOAuthScope = "https://www.googleapis.com/auth/cloud-platform"

// NewVertexTokenSigner uses google.FindDefaultCredentials (metadata, gcloud ADC JSON, GOOGLE_APPLICATION_CREDENTIALS, etc.).
func NewVertexTokenSigner(ctx context.Context) (*VertexTokenSigner, error) {
	if features.SkipVertexAuth() {
		return &VertexTokenSigner{skip: true}, nil
	}
	creds, err := google.FindDefaultCredentials(ctx, vertexOAuthScope)
	if err != nil {
		return nil, fmt.Errorf("vertex signing: default credentials: %w", err)
	}
	if creds.TokenSource == nil {
		return nil, fmt.Errorf("vertex signing: nil TokenSource from default credentials")
	}
	return &VertexTokenSigner{ts: creds.TokenSource}, nil
}

// NewVertexTokenSignerFromSource is for tests or custom token sources.
func NewVertexTokenSignerFromSource(ts oauth2.TokenSource) *VertexTokenSigner {
	return &VertexTokenSigner{ts: ts}
}

// Sign implements CloudRequestSigner.
func (s *VertexTokenSigner) Sign(ctx context.Context, req *http.Request) error {
	if s == nil || s.skip {
		return nil
	}
	tok, err := s.ts.Token()
	if err != nil {
		return fmt.Errorf("vertex signing: access token: %w", err)
	}
	if tok.AccessToken == "" {
		return fmt.Errorf("vertex signing: empty access token")
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	return nil
}
