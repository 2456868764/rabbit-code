package anthropic

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestAuthTransport_APIKey(t *testing.T) {
	var saw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saw = r.Header.Get("x-api-key")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	rt := NewTransportChain(http.DefaultTransport, "sk-test", "")
	c := http.Client{Transport: rt}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if saw != "sk-test" {
		t.Fatal(saw)
	}
}

func TestAuthTransport_Bearer(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	rt := NewTransportChain(http.DefaultTransport, "", "oauth-token")
	c := http.Client{Transport: rt}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if auth != "Bearer oauth-token" {
		t.Fatal(auth)
	}
}

func TestAuthTransport_OAuth401Refresh(t *testing.T) {
	var token atomic.Value
	token.Store("expired")
	var refreshCalls atomic.Int32
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		if r.Header.Get("Authorization") == "Bearer expired" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	rt := NewTransportChainOAuth(http.DefaultTransport, func() string {
		return token.Load().(string)
	}, func(ctx context.Context) error {
		refreshCalls.Add(1)
		token.Store("fresh")
		return nil
	})
	raw := []byte(`{}`)
	req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(raw)), nil
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d auths=%v", resp.StatusCode, auths)
	}
	if refreshCalls.Load() != 1 {
		t.Fatalf("refreshCalls=%d", refreshCalls.Load())
	}
	if len(auths) != 2 || auths[0] != "Bearer expired" || auths[1] != "Bearer fresh" {
		t.Fatalf("%v", auths)
	}
}

func TestClient_PostMessagesStream_OAuth401Refresh(t *testing.T) {
	var token atomic.Value
	token.Store("bad")
	var refreshCalls atomic.Int32
	var roundTrips atomic.Int32
	sse := "data: {\"type\":\"message_stop\"}\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		c := roundTrips.Add(1)
		if c == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, sse)
	}))
	defer srv.Close()
	rt := NewTransportChainOAuth(http.DefaultTransport, func() string {
		return token.Load().(string)
	}, func(ctx context.Context) error {
		refreshCalls.Add(1)
		token.Store("ok")
		return nil
	})
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	body := MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[{"role":"user","content":[{"type":"text","text":"x"}]}]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 2, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	_, _, err = ReadAssistantStream(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if refreshCalls.Load() != 1 || roundTrips.Load() != 2 {
		t.Fatalf("refresh=%d trips=%d", refreshCalls.Load(), roundTrips.Load())
	}
}
