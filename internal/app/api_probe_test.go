package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/api/services"
	"github.com/2456868764/rabbit-code/internal/features"
)

func TestProbeServiceAPI_emptyUsage(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "")
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "")

	rt := &Runtime{}
	resp, err := ProbeServiceAPI(context.Background(), rt, services.EmptyUsage, anthropic.Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if path != "/v1/usage/empty" {
		t.Fatalf("path %q", path)
	}
}

func TestProbeServiceAPI_nilRuntime(t *testing.T) {
	_, err := ProbeServiceAPI(context.Background(), nil, services.EmptyUsage, anthropic.DefaultPolicy())
	if err == nil {
		t.Fatal("expected error")
	}
}
