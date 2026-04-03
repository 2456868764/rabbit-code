package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestNewClientWithPool_PostMessagesStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "k" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "")
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "")

	cl := NewClientWithPool(context.Background(), nil, "k", "")
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	body := MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

func TestResolveAPIOutboundTransport_fallbackOnPartialMTLS(t *testing.T) {
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "/nonexistent/cert.pem")
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "")
	t.Setenv(features.EnvUseBedrock, "")
	rt, err := ResolveAPIOutboundTransport(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from failed full outbound build")
	}
	if rt == nil {
		t.Fatal("nil round tripper")
	}
	_, ok := rt.(*http.Transport)
	if !ok {
		t.Fatalf("fallback should be *http.Transport, got %T", rt)
	}
}
