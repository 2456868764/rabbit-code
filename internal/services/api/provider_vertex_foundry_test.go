package anthropic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestPostMessagesStream_Vertex_streamRawPredict_withProject(t *testing.T) {
	t.Setenv("ANTHROPIC_VERTEX_PROJECT_ID", "test-proj-1")
	t.Setenv("CLOUD_ML_REGION", "us-central1")
	t.Cleanup(func() {
		_ = os.Unsetenv("ANTHROPIC_VERTEX_PROJECT_ID")
		_ = os.Unsetenv("CLOUD_ML_REGION")
	})

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		s := string(b)
		if strings.Contains(s, `"model"`) {
			http.Error(w, "model must not be in Vertex streamRawPredict JSON body", http.StatusBadRequest)
			return
		}
		if !strings.Contains(s, VertexDefaultAnthropicVersion) {
			http.Error(w, "missing anthropic_version", http.StatusBadRequest)
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
	cl.Provider = ProviderVertex
	cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})

	body := MessagesStreamBody{
		Model:     "claude-3-opus@20240229",
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
	// Vertex SDK uses raw model id in the path; url.PathEscape does not encode '@'.
	want := "/projects/test-proj-1/locations/us-central1/publishers/anthropic/models/claude-3-opus@20240229:streamRawPredict"
	if gotPath != want {
		t.Fatalf("path=%q want %q", gotPath, want)
	}
}
