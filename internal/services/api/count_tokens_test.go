package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CountMessagesInputTokens_anthropic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages/count_tokens" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("anthropic-beta") == "" {
			t.Error("expected anthropic-beta header")
		}
		_ = json.NewEncoder(w).Encode(map[string]int{"input_tokens": 42})
	}))
	defer srv.Close()

	c := NewClient(http.DefaultTransport)
	c.BaseURL = srv.URL
	c.Provider = ProviderAnthropic
	msgs, _ := json.Marshal([]map[string]string{{"role": "user", "content": "hi"}})
	n, err := c.CountMessagesInputTokens(context.Background(), "claude-3-5-haiku-20241022", msgs, DefaultPolicy())
	if err != nil || n != 42 {
		t.Fatalf("got %d %v", n, err)
	}
}

func TestClient_CountMessagesInputTokens_bedrockUnsupported(t *testing.T) {
	c := NewClient(http.DefaultTransport)
	c.Provider = ProviderBedrock
	_, err := c.CountMessagesInputTokens(context.Background(), "m", json.RawMessage(`[]`), DefaultPolicy())
	if err != ErrCountTokensUnsupported {
		t.Fatalf("got %v", err)
	}
}
