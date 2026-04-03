package anthropic

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPTransportWithProxyFromEnv(t *testing.T) {
	tr := HTTPTransportWithProxyFromEnv()
	if tr == nil || tr.Proxy == nil {
		t.Fatal("expected non-nil transport and Proxy func")
	}
}

func TestHTTPTransportWithProxyFromEnvAndRoots(t *testing.T) {
	pool := x509.NewCertPool()
	tr := HTTPTransportWithProxyFromEnvAndRoots(pool)
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.RootCAs != pool {
		t.Fatal("expected RootCAs set to pool")
	}
	trNil := HTTPTransportWithProxyFromEnvAndRoots(nil)
	if trNil.TLSClientConfig != nil && trNil.TLSClientConfig.RootCAs != nil {
		t.Fatal("nil pool must not set RootCAs")
	}
}

func TestHTTPTransportWithProxyFromEnv_ReachesServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	cl := &http.Client{Transport: HTTPTransportWithProxyFromEnv()}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := cl.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.Status)
	}
}

func TestNewClient_ProxyAwareRoundTrip(t *testing.T) {
	var sawUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	rt := NewTransportChain(HTTPTransportWithProxyFromEnv(), "k", "")
	cl := NewClient(rt)
	cl.BaseURL = srv.URL
	cl.Provider = ProviderAnthropic
	cl.BetaHeader = MergeBetaHeader([]string{BetaClaudeCode20250219})
	body := MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[]`)}
	resp, err := cl.PostMessagesStream(context.Background(), body, Policy{MaxAttempts: 1, Retry529429: false})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _, err = ReadAssistantStream(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if sawUA == "" {
		t.Fatal("missing User-Agent on proxied transport chain")
	}
}
