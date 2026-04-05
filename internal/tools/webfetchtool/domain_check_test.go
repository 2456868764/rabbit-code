package webfetchtool

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestCheckDomainBlocklist_blocked(t *testing.T) {
	ClearDomainAllowCacheForTest()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"can_fetch":false}`))
	}))
	defer srv.Close()

	err := CheckDomainBlocklist(context.Background(), srv.URL, "example.com", srv.Client())
	if !errors.Is(err, ErrDomainBlocked) {
		t.Fatalf("want ErrDomainBlocked, got %v", err)
	}
}

func TestCheckDomainBlocklist_allowedPositiveCache(t *testing.T) {
	ClearDomainAllowCacheForTest()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"can_fetch":true}`))
	}))
	defer srv.Close()
	c := srv.Client()
	if err := CheckDomainBlocklist(context.Background(), srv.URL, "cached.org", c); err != nil {
		t.Fatal(err)
	}
	if err := CheckDomainBlocklist(context.Background(), srv.URL, "cached.org", c); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 1 {
		t.Fatalf("expected 1 HTTP call, got %d", n.Load())
	}
}
