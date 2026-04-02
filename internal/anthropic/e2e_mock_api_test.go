package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

// E2E: RABBIT_CODE_E2E_MOCK_API=1 (PHASE04_E2E_ACCEPTANCE §2).
func TestE2E_MockAPI_StreamUsage(t *testing.T) {
	if !features.E2EMockAPI() {
		t.Skip("set " + features.EnvE2EMockAPI + "=1")
	}
	sse := "" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"assistant-final\"}}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"usage\":{\"input_tokens\":9,\"output_tokens\":4}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sse)
	}))
	defer srv.Close()

	rt := NewTransportChain(http.DefaultTransport, "e2e-key", "")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})
	body := MessagesStreamBody{Model: "m", MaxTokens: 8, Messages: []byte(`[{"role":"user","content":[{"type":"text","text":"hi"}]}]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	text, u, err := ReadAssistantStream(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if text != "assistant-final" {
		t.Fatalf("got %q", text)
	}
	if u.InputTokens != 9 || u.OutputTokens != 4 {
		t.Fatalf("%+v", u)
	}
}

func TestE2E_OAuthBearer_MockAPI(t *testing.T) {
	if !features.E2EMockAPI() {
		t.Skip("set " + features.EnvE2EMockAPI + "=1")
	}
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()
	rt := NewTransportChain(http.DefaultTransport, "", "tok")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})
	body := MessagesStreamBody{Model: "m", MaxTokens: 8, Messages: []byte(`[{"role":"user","content":[{"type":"text","text":"hi"}]}]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 2, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _, _ = ReadAssistantStream(context.Background(), resp.Body)
	if !strings.HasPrefix(auth, "Bearer ") {
		t.Fatal(auth)
	}
}
