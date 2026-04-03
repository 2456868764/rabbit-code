package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/bootstrap"
	"github.com/2456868764/rabbit-code/internal/cost"
)

func TestPostMessagesStream_MockServer(t *testing.T) {
	sse := "" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"usage\":{\"input_tokens\":3,\"output_tokens\":1}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/v1/messages") {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-key") != "k" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-beta") == "" {
			http.Error(w, "beta", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sse)
	}))
	defer srv.Close()

	rt := NewTransportChain(http.DefaultTransport, "k", "")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})

	body := MessagesStreamBody{
		Model:     "claude-3-5-haiku-20241022",
		MaxTokens: 256,
		Messages:  []byte(`[{"role":"user","content":[{"type":"text","text":"x"}]}]`),
	}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 3, Retry529429: true})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	text, u, err := ReadAssistantStream(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hi" {
		t.Fatalf("text=%q", text)
	}
	if u.InputTokens != 3 || u.OutputTokens != 1 {
		t.Fatalf("%+v", u)
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}

	st := bootstrap.NewState()
	cost.ApplyUsageToBootstrap(st, cost.Usage{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
	})
	if st.LastTokenUsage().InputTokens != 3 {
		t.Fatal(st.LastTokenUsage())
	}
}
