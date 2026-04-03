package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestParseCurlStyleHeaders(t *testing.T) {
	h := ParseCurlStyleHeaders("X-Foo: a\nX-Bar: b\n")
	if h.Get("X-Foo") != "a" || h.Get("X-Bar") != "b" {
		t.Fatal(h)
	}
}

func TestMergedCustomHeadersFromEnv_RabbitOverrides(t *testing.T) {
	t.Setenv(EnvAnthropicCustomHeaders, "X-A: 1\nX-B: 2")
	t.Setenv(EnvRabbitAnthropicCustomHeaders, "X-B: 3")
	h := MergedCustomHeadersFromEnv()
	if h.Get("X-A") != "1" || h.Get("X-B") != "3" {
		t.Fatal(h)
	}
}

func TestMergedCustomHeadersFromEnv_Empty(t *testing.T) {
	t.Setenv(EnvAnthropicCustomHeaders, "")
	t.Setenv(EnvRabbitAnthropicCustomHeaders, "")
	if len(MergedCustomHeadersFromEnv()) != 0 {
		t.Fatal()
	}
}

func TestPostMessagesStream_OAuthBetaAppendAndAntiDistillationHeaders(t *testing.T) {
	t.Setenv(features.EnvOAuthBetaAppend, "oauth-append-test-beta")
	t.Setenv(features.EnvAntiDistillation, "1")
	t.Setenv(features.EnvAntiDistillationHeader, "X-Test-Anti-Distill")
	t.Setenv(features.EnvAntiDistillationValue, "on")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		beta := r.Header.Get("anthropic-beta")
		if !strings.Contains(beta, "oauth-append-test-beta") {
			http.Error(w, beta, http.StatusBadRequest)
			return
		}
		if r.Header.Get("X-Test-Anti-Distill") != "on" {
			http.Error(w, "ad", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	rt := NewTransportChain(http.DefaultTransport, "k", "")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	cl.BetaHeader = MergeBetaHeader([]string{BetaWebSearch})
	body := MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}
