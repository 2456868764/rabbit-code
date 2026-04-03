package querydeps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/features"
)

func TestAnthropicAssistant_nilClient(t *testing.T) {
	a := &AnthropicAssistant{Client: nil}
	_, err := a.StreamAssistant(context.Background(), "m", 1, []byte(`[]`))
	if err != ErrNilAnthropicClient {
		t.Fatalf("got %v", err)
	}
}

func TestAnthropicAssistant_httptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "k" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 8,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
	}
	msgs := []byte(`[{"role":"user","content":[{"type":"text","text":"yo"}]}]`)
	text, err := a.StreamAssistant(context.Background(), "", 0, msgs)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hi" {
		t.Fatalf("got %q", text)
	}
}

func TestAnthropicAssistant_StreamAssistant_promptCacheBreakFromContext(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	var hookCalls int
	ctx := ContextWithOnPromptCacheBreak(context.Background(), func() { hookCalls++ })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"error\",\"message\":\"cache_break\"}\n\n")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 8,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
	}
	msgs := []byte(`[{"role":"user","content":[{"type":"text","text":"yo"}]}]`)
	_, err := a.StreamAssistant(ctx, "", 0, msgs)
	if err == nil {
		t.Fatal("expected error")
	}
	if hookCalls != 1 {
		t.Fatalf("prompt cache break hook: want 1 call, got %d", hookCalls)
	}
}

func TestAnthropicAssistant_AssistantTurn_promptCacheBreakFromContext(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	var hookCalls int
	ctx := ContextWithOnPromptCacheBreak(context.Background(), func() { hookCalls++ })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"error\",\"message\":\"cache_break\"}\n\n")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 8,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
	}
	msgs := []byte(`[{"role":"user","content":[{"type":"text","text":"yo"}]}]`)
	_, err := a.AssistantTurn(ctx, "", 0, msgs)
	if err == nil {
		t.Fatal("expected error")
	}
	if hookCalls != 1 {
		t.Fatalf("prompt cache break hook (AssistantTurn path): want 1 call, got %d", hookCalls)
	}
}

func TestAnthropicAssistant_streamBody_anthropicBetaCachedMicrocompact(t *testing.T) {
	t.Setenv(features.EnvCachedMicrocompact, "true")
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")

	okCh := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			okCh <- false
			http.Error(w, err.Error(), 500)
			return
		}
		var body struct {
			AnthropicBeta []string `json:"anthropic_beta"`
		}
		if err := json.Unmarshal(b, &body); err != nil {
			okCh <- false
			http.Error(w, err.Error(), 400)
			return
		}
		ok := len(body.AnthropicBeta) == 1 && body.AnthropicBeta[0] == anthropic.BetaCachedMicrocompactBody
		okCh <- ok
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"z\"}}\n\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 8,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
	}
	msgs := []byte(`[{"role":"user","content":[{"type":"text","text":"yo"}]}]`)
	_, err := a.StreamAssistant(context.Background(), "", 0, msgs)
	if err != nil {
		t.Fatal(err)
	}
	if !<-okCh {
		t.Fatal("expected JSON anthropic_beta with BetaCachedMicrocompactBody when RABBIT_CODE_CACHED_MICROCOMPACT is on")
	}
}

func TestStreamAssistantFunc(t *testing.T) {
	var f StreamAssistantFunc = func(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
		return fmt.Sprintf("%s:%d:%s", model, maxTokens, string(messagesJSON)), nil
	}
	out, err := f.StreamAssistant(context.Background(), "x", 3, []byte(`[]`))
	if err != nil || out != "x:3:[]" {
		t.Fatalf("%q %v", out, err)
	}
}
