package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRequest_UnattendedPersistent_429EventuallyOK(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) <= 4 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	pol := Policy{MaxAttempts: 2, Retry529429: true, Unattended: true}
	resp, err := DoRequest(context.Background(), http.DefaultTransport, req, pol)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}
	if n.Load() < 5 {
		t.Fatalf("attempts=%d", n.Load())
	}
	_, _ = io.ReadAll(resp.Body)
}

func TestDoRequest_UnattendedPersistent_Exhausted529BudgetThenOK(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) < 2 {
			w.WriteHeader(529)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	pol := Policy{
		MaxAttempts:                 1,
		Retry529429:                 true,
		Unattended:                  true,
		InitialConsecutive529Errors: Max529Retries,
	}
	resp, err := DoRequest(context.Background(), http.DefaultTransport, req, pol)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.StatusCode)
	}
	_, _ = io.ReadAll(resp.Body)
}

func TestDoRequest_UnattendedPersistent_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	pol := Policy{MaxAttempts: 1, Retry529429: true, Unattended: true}
	_, err := DoRequest(ctx, http.DefaultTransport, req, pol)
	if err == nil || err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}

func TestDoRequest_Persistent503Returns(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := n.Add(1)
		if c == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	pol := Policy{MaxAttempts: 1, Retry529429: true, Unattended: true}
	_, err := DoRequest(context.Background(), http.DefaultTransport, req, pol)
	if err == nil {
		t.Fatal("expected error")
	}
}
