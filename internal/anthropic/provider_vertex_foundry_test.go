package anthropic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMessagesPath_VertexFoundryAnthropic(t *testing.T) {
	if p := MessagesPath(ProviderAnthropic); p != "/v1/messages" {
		t.Fatal(p)
	}
	if p := MessagesPath(ProviderVertex); p != "/v1/messages" {
		t.Fatal(p)
	}
	if p := MessagesPath(ProviderFoundry); p != "/v1/messages" {
		t.Fatal(p)
	}
}

func TestPostMessagesStream_VertexAndFoundryPath(t *testing.T) {
	for _, prov := range []Provider{ProviderVertex, ProviderFoundry} {
		name := fmt.Sprintf("%v", prov)
		t.Run(name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				b, _ := io.ReadAll(r.Body)
				if len(b) == 0 {
					http.Error(w, "body", http.StatusBadRequest)
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
			cl.Provider = prov
			cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})
			body := MessagesStreamBody{
				Model:     "m",
				MaxTokens: 8,
				Messages:  []byte(`[{"role":"user","content":[{"type":"text","text":"x"}]}]`),
			}
			resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			_, _, err = ReadAssistantStream(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if gotPath != "/v1/messages" {
				t.Fatalf("path=%q", gotPath)
			}
		})
	}
}
