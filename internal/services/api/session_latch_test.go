package anthropic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestSessionLatchedBetas_mergeIntoHeader(t *testing.T) {
	t.Setenv(features.EnvTranscriptClassifier, "1")

	var beta string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		beta = r.Header.Get("anthropic-beta")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	cl := NewClient(http.DefaultTransport)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	cl.LatchFastModeHeader()
	cl.LatchAFKHeader()
	pol := Policy{MaxAttempts: 1, Retry529429: false, AgenticQuery: true, QuerySource: QuerySourceReplMainThread}
	cl.LatchCacheEditingHeader()

	body := MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, pol)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if !strings.Contains(beta, BetaFastMode) || !strings.Contains(beta, BetaAFKMode) || !strings.Contains(beta, BetaPromptCachingScope) {
		t.Fatalf("anthropic-beta=%q", beta)
	}
}
