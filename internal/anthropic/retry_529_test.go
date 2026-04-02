package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestDoRequest_529Exhausted(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(529)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), http.DefaultTransport, req, Policy{MaxAttempts: 20, Retry529429: true})
	if err == nil {
		t.Fatal("expected error")
	}
	// First 529 + 3 retries = 4 round trips before giving up on 529.
	if n.Load() != 4 {
		t.Fatalf("round trips=%d", n.Load())
	}
}

func TestDoRequest_429NotLimitedBy529Budget(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := n.Add(1)
		if c < 5 {
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := DoRequest(context.Background(), http.DefaultTransport, req, Policy{MaxAttempts: 10, Retry529429: true})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}
	if n.Load() != 5 {
		t.Fatalf("got %d", n.Load())
	}
}

func TestDoRequest_QuerySourceNo529(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(529)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), http.DefaultTransport, req, Policy{
		MaxAttempts:   20,
		Retry529429: true,
		QuerySource: QuerySourceNo529,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 1 {
		t.Fatalf("round trips=%d want 1", n.Load())
	}
}

func TestDoRequest_StrictForeground529_unknownSource(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(529)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), http.DefaultTransport, req, Policy{
		MaxAttempts:           20,
		Retry529429:         true,
		QuerySource:         QuerySource("title_suggestion"),
		StrictForeground529: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 1 {
		t.Fatalf("round trips=%d want 1 (no 529 retry for non-foreground source)", n.Load())
	}
}

func TestDoRequest_StrictForeground529_sdkStillRetries529(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(529)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), http.DefaultTransport, req, Policy{
		MaxAttempts:           20,
		Retry529429:         true,
		QuerySource:         QuerySourceSDK,
		StrictForeground529: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 4 {
		t.Fatalf("round trips=%d want 4", n.Load())
	}
}

func TestDoRequest_InitialConsecutive529Errors(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(529)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), http.DefaultTransport, req, Policy{
		MaxAttempts:                 20,
		Retry529429:               true,
		InitialConsecutive529Errors: 2,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// Max529Retries=3, initial=2 → one 529 slot left → 2 total responses.
	if n.Load() != 2 {
		t.Fatalf("round trips=%d want 2", n.Load())
	}
}

func TestDefaultPolicy_StrictForeground529FromEnv(t *testing.T) {
	t.Setenv(features.EnvStrictForeground529, "1")
	p := DefaultPolicy()
	if !p.StrictForeground529 {
		t.Fatal("expected StrictForeground529 from env")
	}
	t.Setenv(features.EnvStrictForeground529, "")
	p2 := DefaultPolicy()
	if p2.StrictForeground529 {
		t.Fatal("expected default off when env empty")
	}
}

func TestDoRequest_StrictForeground529_emptySourceRetries529(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.WriteHeader(529)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := DoRequest(context.Background(), http.DefaultTransport, req, Policy{
		MaxAttempts:           20,
		Retry529429:         true,
		QuerySource:         QuerySourceDefault,
		StrictForeground529: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 4 {
		t.Fatalf("round trips=%d want 4 (undefined / default tags retry 529 in TS)", n.Load())
	}
}
