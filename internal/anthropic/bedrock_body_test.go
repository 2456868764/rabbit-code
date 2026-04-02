package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostMessagesStream_BedrockAnthropicBetaInBody(t *testing.T) {
	sse := "data: {\"type\":\"message_stop\"}\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var probe struct {
			AnthropicBeta []string `json:"anthropic_beta"`
			Stream        bool     `json:"stream"`
		}
		if err := json.Unmarshal(b, &probe); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(probe.AnthropicBeta) != 1 || probe.AnthropicBeta[0] != BetaInterleavedThinking {
			http.Error(w, "missing body beta", http.StatusBadRequest)
			return
		}
		h := r.Header.Get("anthropic-beta")
		if strings.Contains(h, BetaInterleavedThinking) {
			http.Error(w, "interleaved must not be in header", http.StatusBadRequest)
			return
		}
		if !strings.Contains(h, BetaWebSearch) {
			http.Error(w, "want web_search in header", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sse))
	}))
	defer srv.Close()

	rt := NewTransportChain(http.DefaultTransport, "k", "")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderBedrock
	cl.SetBetaNames([]string{BetaInterleavedThinking, BetaWebSearch})

	body := MessagesStreamBody{
		Model:     "m",
		MaxTokens: 8,
		Messages:  []byte(`[{"role":"user","content":[{"type":"text","text":"x"}]}]`),
	}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 2, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _, _ = ReadAssistantStream(context.Background(), resp.Body)
}

func TestPostMessagesStream_BedrockExplicitBodyBetasPreserved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"anthropic_beta":["`+BetaContext1M+`"]`) &&
			!strings.Contains(string(b), BetaContext1M) {
			http.Error(w, string(b), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	rt := NewTransportChain(http.DefaultTransport, "k", "")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderBedrock
	cl.BetaHeader = MergeBetaHeader([]string{BetaWebSearch})

	body := MessagesStreamBody{
		Model:         "m",
		MaxTokens:     8,
		Messages:      []byte(`[]`),
		AnthropicBeta: []string{BetaContext1M},
	}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}
