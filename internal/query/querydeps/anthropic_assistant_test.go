package querydeps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

type testMicrocompactMarker struct {
	marked bool
}

func (t *testMicrocompactMarker) MarkToolsSentToAPIState() { t.marked = true }

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
		ok := false
		for _, x := range body.AnthropicBeta {
			if x == anthropic.BetaCachedMicrocompactBody {
				ok = true
				break
			}
		}
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

func TestAnthropicAssistant_streamBody_contextManagement_ant(t *testing.T) {
	t.Setenv(features.EnvUserType, "ant")
	t.Setenv(features.EnvUserTypeRabbit, "")
	t.Setenv(features.EnvUseAPIContextManagement, "1")
	t.Setenv(features.EnvUseAPIClearToolResults, "1")
	t.Setenv(features.EnvUseAPIClearToolUses, "")
	t.Setenv(features.EnvAPIMaxInputTokens, "")
	t.Setenv(features.EnvAPITargetInputTokens, "")
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
			AnthropicBeta     []string        `json:"anthropic_beta"`
			ContextManagement json.RawMessage `json:"context_management"`
		}
		if err := json.Unmarshal(b, &body); err != nil {
			okCh <- false
			http.Error(w, err.Error(), 400)
			return
		}
		var cm struct {
			Edits []json.RawMessage `json:"edits"`
		}
		_ = json.Unmarshal(body.ContextManagement, &cm)
		hasBeta := false
		for _, x := range body.AnthropicBeta {
			if x == anthropic.BetaContextManagement {
				hasBeta = true
				break
			}
		}
		okCh <- hasBeta && len(cm.Edits) > 0
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
	_, err := a.StreamAssistant(context.Background(), "claude-3-5-haiku-20241022", 0, msgs)
	if err != nil {
		t.Fatal(err)
	}
	if !<-okCh {
		t.Fatal("expected context_management + context-management beta for ant + USE_API_CONTEXT_MANAGEMENT")
	}
}

func TestAnthropicAssistant_streamBody_contextManagement_model4Thinking(t *testing.T) {
	t.Setenv(features.EnvUserType, "")
	t.Setenv(features.EnvUserTypeRabbit, "")
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
			ContextManagement json.RawMessage `json:"context_management"`
		}
		if err := json.Unmarshal(b, &body); err != nil {
			okCh <- false
			http.Error(w, err.Error(), 400)
			return
		}
		var cm map[string]interface{}
		_ = json.Unmarshal(body.ContextManagement, &cm)
		_, ok := cm["edits"]
		okCh <- ok
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	opts := compact.APIContextManagementOptions{HasThinking: true}
	a := &AnthropicAssistant{
		Client:                   cl,
		DefaultModel:             "m",
		DefaultMaxTokens:         8,
		Policy:                   anthropic.Policy{MaxAttempts: 1, Retry529429: false},
		APIContextManagementOpts: &opts,
	}
	msgs := []byte(`[{"role":"user","content":[{"type":"text","text":"yo"}]}]`)
	_, err := a.StreamAssistant(context.Background(), "claude-sonnet-4-20250514", 0, msgs)
	if err != nil {
		t.Fatal(err)
	}
	if !<-okCh {
		t.Fatal("expected context_management for Claude 4 + HasThinking")
	}
}

func TestAnthropicAssistant_MicrocompactBuffer_markAfterStreamSuccess(t *testing.T) {
	t.Setenv(features.EnvCachedMicrocompact, "true")
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	var buf testMicrocompactMarker
	a := &AnthropicAssistant{
		Client:             cl,
		DefaultModel:       "m",
		DefaultMaxTokens:   8,
		Policy:             anthropic.Policy{MaxAttempts: 1, Retry529429: false},
		MicrocompactBuffer: &buf,
	}
	msgs := []byte(`[{"role":"user","content":[{"type":"text","text":"yo"}]}]`)
	_, err := a.StreamAssistant(context.Background(), "", 0, msgs)
	if err != nil {
		t.Fatal(err)
	}
	if !buf.marked {
		t.Fatal("expected MarkToolsSentToAPIState after successful stream when CACHED_MICROCOMPACT is on")
	}
}

func TestAnthropicAssistant_StreamCompactSummary_success(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")

	var gotTools, gotSys string
	var gotMsgs []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			System   json.RawMessage `json:"system"`
			Messages json.RawMessage `json:"messages"`
			Tools    json.RawMessage `json:"tools"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotTools = string(body.Tools)
		_ = json.Unmarshal(body.System, &gotSys)
		gotMsgs = append([]byte(nil), body.Messages...)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"<summary>done</summary>\"}}\n\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	var buf testMicrocompactMarker
	a := &AnthropicAssistant{
		Client:             cl,
		DefaultModel:       "m",
		DefaultMaxTokens:   8,
		Policy:             anthropic.Policy{MaxAttempts: 1, Retry529429: false},
		MicrocompactBuffer: &buf,
	}
	msgs := []byte(`[{"role":"user","content":[{"type":"text","text":"yo"}]}]`)
	out, err := a.StreamCompactSummary(context.Background(), msgs, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotTools, "Read") {
		t.Fatalf("expected compact tools Read, got %q", gotTools)
	}
	if !strings.Contains(gotSys, "summarizing conversations") {
		t.Fatalf("system: %q", gotSys)
	}
	if !strings.Contains(string(gotMsgs), "CRITICAL: Respond with TEXT ONLY") {
		t.Fatalf("messages: %s", gotMsgs)
	}
	if !strings.Contains(out, "done") {
		t.Fatalf("got %q", out)
	}
	if buf.marked {
		t.Fatal("compact summary stream should not mark microcompact buffer")
	}
}

func TestAnthropicAssistant_StreamCompactSummary_forkPath(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	t.Setenv(features.EnvCompactCachePrefix, "1")

	var forked bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("streaming should not be called when fork succeeds")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 256,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
		ForkCompactSummary: func(ctx context.Context, summaryUserJSON []byte, transcriptJSON []byte) (string, error) {
			forked = true
			return "<summary>forked</summary>", nil
		},
	}
	out, err := a.StreamCompactSummary(context.Background(), []byte(`[{"role":"user","content":[{"type":"text","text":"a"}]}]`), "")
	if err != nil {
		t.Fatal(err)
	}
	if !forked {
		t.Fatal("expected fork")
	}
	if !strings.Contains(out, "forked") {
		t.Fatalf("%q", out)
	}
}

func TestAnthropicAssistant_StreamCompactSummary_PTLRetryThenOK(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		text := "<summary>second</summary>"
		if calls == 1 {
			text = "Prompt is too long: 100 tokens > 50 maximum"
		}
		_, _ = io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\""+strings.ReplaceAll(text, `"`, `\"`)+"\"}}\n\n")
		_, _ = io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 8192,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
	}
	msgs := []byte(`[
		{"role":"user","content":[{"type":"text","text":"u1"}]},
		{"role":"assistant","id":"r1","content":[{"type":"text","text":"a1"}]},
		{"role":"user","content":[{"type":"text","text":"u2"}]},
		{"role":"assistant","id":"r2","content":[{"type":"text","text":"a2"}]}
	]`)
	out, err := a.StreamCompactSummary(context.Background(), msgs, "")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("want 2 API calls after PTL retry, got %d", calls)
	}
	if !strings.Contains(out, "second") {
		t.Fatalf("got %q", out)
	}
}

func TestStreamingCompactExecutor_nilClient(t *testing.T) {
	ex := StreamingCompactExecutor(nil, "")
	_, _, err := ex(context.Background(), compact.RunExecuting, []byte(`[]`))
	if err != ErrNilAnthropicClient {
		t.Fatalf("got %v", err)
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
