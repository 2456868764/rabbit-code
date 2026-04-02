package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/anthropic/services"
	"github.com/2456868764/rabbit-code/internal/features"
)

func TestRunProbe_unknownModule(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:9")
	err := RunProbe(context.Background(), io.Discard, "nope.ts")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunProbe_emptyUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/usage/empty" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "")
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "")

	var buf bytes.Buffer
	if err := RunProbe(context.Background(), &buf, services.EmptyUsage); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "204") && !strings.Contains(buf.String(), "No Content") {
		t.Fatalf("output %q", buf.String())
	}
}
