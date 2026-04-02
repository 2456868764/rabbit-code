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

func TestPostMessagesStreamReadAssistant_OnStreamUsage(t *testing.T) {
	sse := "" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Z\"}}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":2}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/v1/messages") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, sse)
	}))
	defer srv.Close()

	var hookCalls int
	st := bootstrap.NewState()
	rt := NewTransportChain(http.DefaultTransport, "k", "")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})
	cl.OnStreamUsage = func(u UsageDelta) {
		hookCalls++
		cost.ApplyUsageToBootstrap(st, cost.FromUsageDelta(
			u.InputTokens, u.CacheCreationInputTokens, u.CacheReadInputTokens, u.OutputTokens,
		))
	}

	body := MessagesStreamBody{
		Model:     "m",
		MaxTokens: 8,
		Messages:  []byte(`[]`),
	}
	text, u, err := cl.PostMessagesStreamReadAssistant(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	if text != "Z" {
		t.Fatalf("text=%q", text)
	}
	if u.InputTokens != 5 || u.OutputTokens != 2 {
		t.Fatalf("%+v", u)
	}
	if hookCalls != 1 {
		t.Fatalf("hookCalls=%d", hookCalls)
	}
	if st.LastTokenUsage().InputTokens != 5 || st.LastTokenUsage().OutputTokens != 2 {
		t.Fatal(st.LastTokenUsage())
	}
}
