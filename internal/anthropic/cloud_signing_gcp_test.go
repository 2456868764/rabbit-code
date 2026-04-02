package anthropic

import (
	"context"
	"net/http"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"golang.org/x/oauth2"
)

func TestVertexTokenSigner_Sign_setsBearer(t *testing.T) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "unit-test-token"})
	s := NewVertexTokenSignerFromSource(ts)
	req, _ := http.NewRequest(http.MethodPost, "https://us-central1-aiplatform.googleapis.com/v1/projects/p/locations/us-central1/publishers/anthropic/models/m:streamRawPredict", nil)
	if err := s.Sign(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if g, w := req.Header.Get("Authorization"), "Bearer unit-test-token"; g != w {
		t.Fatalf("Authorization=%q want %q", g, w)
	}
}

func TestVertexTokenSigner_skipNoops(t *testing.T) {
	t.Setenv(features.EnvSkipVertexAuth, "1")
	s, err := NewVertexTokenSigner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err := s.Sign(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatal("expected no auth when skip")
	}
}
