package query

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestStreamingCompactExecutorWithConfig_hooksAndNext(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"<summary>x</summary>"}}` + "\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &anthropic.AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 256,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
	}
	var pre, post bool
	ex := StreamingCompactExecutorWithConfig(a, StreamingCompactExecutorConfig{
		CustomInstructions:        "from-hook",
		ReturnNextTranscript:      true,
		SuppressFollowUpQuestions: true,
		PreCompactHook: func(ctx context.Context, auto bool) (string, string, error) {
			pre = true
			if !auto {
				t.Fatal("expected auto from context")
			}
			return "extra", "pre-msg", nil
		},
		PostCompactHook: func(ctx context.Context, auto bool, raw string) (string, error) {
			post = true
			return "post-msg", nil
		},
		SessionStartHook: func(ctx context.Context) ([]json.RawMessage, error) {
			return []json.RawMessage{json.RawMessage(`{"type":"system","content":"h"}`)}, nil
		},
	})
	ctx := compact.ContextWithExecutorSuggestMeta(context.Background(), compact.ExecutorSuggestMeta{AutoCompact: true})
	sum, next, err := ex(ctx, compact.RunExecuting, []byte(`[{"uuid":"z","role":"user","content":[{"type":"text","text":"a"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if !pre || !post {
		t.Fatalf("hooks pre=%v post=%v", pre, post)
	}
	if !strings.Contains(sum, "pre-msg") || !strings.Contains(sum, "post-msg") {
		t.Fatalf("display prefix: %q", sum)
	}
	if !strings.Contains(string(next), "compact_boundary") {
		t.Fatalf("next: %s", next)
	}
}

func TestStreamingCompactExecutorWithConfig_postCompactAttachments(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"<summary>z</summary>"}}` + "\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	cl := anthropic.NewClient(anthropic.NewTransportChain(http.DefaultTransport, "k", ""))
	cl.BaseURL = srv.URL
	cl.Provider = anthropic.ProviderAnthropic

	a := &anthropic.AnthropicAssistant{
		Client:           cl,
		DefaultModel:     "m",
		DefaultMaxTokens: 256,
		Policy:           anthropic.Policy{MaxAttempts: 1, Retry529429: false},
	}
	ex := StreamingCompactExecutorWithConfig(a, StreamingCompactExecutorConfig{
		ReturnNextTranscript: true,
		PostCompactAttachmentsJSON: func(ctx context.Context, transcriptBefore []byte, raw string) ([]json.RawMessage, error) {
			rawAtt, err := compact.CreateAttachmentMessageJSON(map[string]interface{}{
				"type": "invoked_skills",
				"skills": []interface{}{
					map[string]string{"name": "n", "path": "/p", "content": "c"},
				},
			})
			if err != nil {
				return nil, err
			}
			return []json.RawMessage{rawAtt}, nil
		},
	})
	ctx := context.Background()
	_, next, err := ex(ctx, compact.RunExecuting, []byte(`[{"role":"user","content":[{"type":"text","text":"a"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(next), "invoked_skills") {
		t.Fatalf("next: %s", next)
	}
}
