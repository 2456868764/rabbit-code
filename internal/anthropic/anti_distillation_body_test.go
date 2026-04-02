package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestPostMessagesStream_AntiDistillationFakeToolsBody(t *testing.T) {
	t.Setenv(features.EnvAntiDistillation, "1")
	t.Setenv(features.EnvAntiDistillationFakeTools, "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Anti []string `json:"anti_distillation"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		_ = r.Body.Close()
		if len(payload.Anti) != 1 || payload.Anti[0] != "fake_tools" {
			http.Error(w, "no anti_distillation", http.StatusBadRequest)
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
	body := MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

func TestMarshalMessagesStreamJSON_vertexIncludesAntiDistillation(t *testing.T) {
	t.Setenv("ANTHROPIC_VERTEX_BASE_URL", "")
	t.Setenv("ANTHROPIC_VERTEX_PROJECT_ID", "p1")
	t.Setenv("CLOUD_ML_REGION", "us-central1")
	t.Setenv(features.EnvAntiDistillation, "1")
	t.Setenv(features.EnvAntiDistillationFakeTools, "1")

	cl := NewClient(http.DefaultTransport)
	cl.Provider = ProviderVertex
	cl.BaseURL = "https://example.com"
	raw, err := cl.marshalMessagesStreamJSON(MessagesStreamBody{
		Model: "m", MaxTokens: 1, Messages: []byte(`[]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"anti_distillation":["fake_tools"]`)) {
		t.Fatalf("body=%s", raw)
	}
}
