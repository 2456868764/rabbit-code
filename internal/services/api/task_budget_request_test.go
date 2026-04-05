package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestPostMessagesStream_TaskBudgetJSONAndBeta(t *testing.T) {
	t.Setenv(features.EnvHTTPUserAgent, "")
	sse := "data: {\"type\":\"message_stop\"}\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var probe struct {
			OutputConfig struct {
				TaskBudget struct {
					Type  string `json:"type"`
					Total int    `json:"total"`
				} `json:"task_budget"`
			} `json:"output_config"`
		}
		if err := json.Unmarshal(b, &probe); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if probe.OutputConfig.TaskBudget.Type != "tokens" || probe.OutputConfig.TaskBudget.Total != 10000 {
			http.Error(w, string(b), http.StatusBadRequest)
			return
		}
		beta := r.Header.Get("anthropic-beta")
		if beta == "" || !strings.Contains(beta, BetaTaskBudgets) {
			http.Error(w, "missing task budget beta: "+beta, http.StatusBadRequest)
			return
		}
		if r.Header.Get("X-Claude-Code-Session-Id") != "sess-xyz" {
			http.Error(w, "session", http.StatusBadRequest)
			return
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			http.Error(w, "anthropic-version", http.StatusBadRequest)
			return
		}
		if r.Header.Get("User-Agent") != features.DefaultHTTPUserAgent {
			http.Error(w, "user-agent: "+r.Header.Get("User-Agent"), http.StatusBadRequest)
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
	cl.Provider = ProviderAnthropic
	cl.SessionID = "sess-xyz"
	cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})
	rem := 9000
	body := MessagesStreamBody{
		Model:     "m",
		MaxTokens: 8,
		Messages:  []byte(`[]`),
		OutputConfig: &OutputConfig{
			TaskBudget: &TaskBudgetParam{Type: "tokens", Total: 10000, Remaining: &rem},
		},
	}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}
