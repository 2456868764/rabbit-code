package webfetchtool

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Avoid real domain_info calls in unit tests.
	_ = os.Setenv("RABBIT_CODE_SKIP_WEBFETCH_PREFLIGHT", "1")
	os.Exit(m.Run())
}

func tlsServer(h http.HandlerFunc) (*httptest.Server, *http.Client) {
	s := httptest.NewTLSServer(h)
	c := s.Client()
	c.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	c.Timeout = 60 * time.Second
	return s, c
}

func TestWebFetch_plainText(t *testing.T) {
	srv, client := tlsServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("hello world"))
	})
	defer srv.Close()

	rc := &RunContext{HTTPClient: client}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"summarize"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m.Code != 200 {
		t.Fatalf("code %d", m.Code)
	}
	if m.Result == "" || len(m.Result) < 5 {
		t.Fatalf("unexpected result: %q", m.Result)
	}
	if m.URL != srv.URL {
		t.Fatalf("url field want %q got %q", srv.URL, m.URL)
	}
}

func TestWebFetch_crossHostRedirect(t *testing.T) {
	other, _ := tlsServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	defer other.Close()

	srv, client := tlsServer(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusFound)
	})
	defer srv.Close()

	rc := &RunContext{HTTPClient: client}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m.Code != http.StatusFound {
		t.Fatalf("code %d", m.Code)
	}
	if m.Result == "" {
		t.Fatal("empty result")
	}
}

func TestWebFetch_sameHostRedirectThenBody(t *testing.T) {
	srv, client := tlsServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/end" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("final"))
			return
		}
		http.Redirect(w, r, "/end", http.StatusFound)
	})
	defer srv.Close()

	rc := &RunContext{HTTPClient: client}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"p"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m.Result == "" {
		t.Fatalf("empty result")
	}
}

func TestWebFetch_applyPrompt(t *testing.T) {
	srv, client := tlsServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("data"))
	})
	defer srv.Close()

	rc := &RunContext{
		HTTPClient: client,
		ApplyPrompt: func(_ context.Context, markdown, prompt string, _, _ bool) (string, error) {
			return "OUT:" + markdown + ":" + prompt, nil
		},
	}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"q"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m.Result != "OUT:data:q" {
		t.Fatalf("%q", m.Result)
	}
}

func TestWebFetch_urlCacheSecondHitNoNetwork(t *testing.T) {
	ClearURLCacheForTest()
	var n atomic.Int32
	srv, client := tlsServer(func(w http.ResponseWriter, _ *http.Request) {
		n.Add(1)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("cached-body"))
	})
	defer srv.Close()

	rc := &RunContext{HTTPClient: client}
	ctx := WithRunContext(context.Background(), rc)
	url := srv.URL
	payload := []byte(`{"url":` + jsonString(url) + `,"prompt":"a"}`)
	if _, err := New().Run(ctx, payload); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 1 {
		t.Fatalf("first fetch count %d", n.Load())
	}
	if _, err := New().Run(ctx, payload); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 1 {
		t.Fatalf("cache should skip second GET, got count %d", n.Load())
	}
}

func TestWebFetch_binaryPersistNote(t *testing.T) {
	ClearURLCacheForTest()
	dir := t.TempDir()
	srv, client := tlsServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 minimal"))
	})
	defer srv.Close()

	rc := &RunContext{HTTPClient: client, ToolResultsDir: dir}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"what"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m.Result == "" {
		t.Fatal("empty")
	}
	if !strings.Contains(m.Result, "also saved to") {
		t.Fatalf("missing persist note: %s", m.Result)
	}
}

func TestMapWebFetchToolResultForMessagesAPI(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"result": "hello"})
	if s := MapWebFetchToolResultForMessagesAPI(raw); s != "hello" {
		t.Fatalf("%q", s)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

