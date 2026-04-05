package webfetchtool

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebFetch_plainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	out, err := New().Run(context.Background(), []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"summarize"}`))
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
	if !strings.Contains(m.Result, "hello world") {
		t.Fatalf("missing body in result: %q", m.Result)
	}
	if m.URL != srv.URL {
		t.Fatalf("url field want %q got %q", srv.URL, m.URL)
	}
}

func TestWebFetch_crossHostRedirect(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer other.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusFound)
	}))
	defer srv.Close()

	out, err := New().Run(context.Background(), []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"x"}`))
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
	if !strings.Contains(m.Result, "REDIRECT DETECTED") || !strings.Contains(m.Result, other.URL) {
		t.Fatalf("unexpected result: %s", m.Result)
	}
}

func TestWebFetch_sameHostRedirectThenBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/end" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("final"))
			return
		}
		http.Redirect(w, r, "/end", http.StatusFound)
	}))
	defer srv.Close()

	out, err := New().Run(context.Background(), []byte(`{"url":`+jsonString(srv.URL)+`,"prompt":"p"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.Result, "final") {
		t.Fatalf("%q", m.Result)
	}
}

func TestWebFetch_applyPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	rc := &RunContext{
		ApplyPrompt: func(_ context.Context, markdown, prompt string, _ bool) (string, error) {
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
