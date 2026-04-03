package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/bootstrap"
	"github.com/2456868764/rabbit-code/internal/features"
)

func TestNewAnthropicClient_nilRuntime(t *testing.T) {
	if NewAnthropicClient(context.Background(), nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestNewAnthropicClient_sessionAndAuthHeaders(t *testing.T) {
	var sawSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawSession = r.Header.Get("X-Claude-Code-Session-Id")
		if r.Header.Get("x-api-key") != "k" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("RABBIT_CODE_API_KEY", "")
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "")
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "")

	st := bootstrap.NewState()
	st.SetSessionID("sess-test-api")

	rt := &Runtime{
		State:           st,
		GlobalConfigDir: t.TempDir(),
	}
	cl := NewAnthropicClient(context.Background(), rt)
	if cl == nil {
		t.Fatal("nil client")
	}
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic
	body := anthropic.MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, anthropic.Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if sawSession != "sess-test-api" {
		t.Fatalf("session header: got %q", sawSession)
	}
}
