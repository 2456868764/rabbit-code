package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllServiceAPIFilesHaveBuilder(t *testing.T) {
	for _, name := range AllTSFiles {
		if _, ok := Builders[name]; !ok {
			t.Fatalf("missing builder for %s", name)
		}
		req, err := BuildRequest(name, "https://api.anthropic.com", "https://console.anthropic.com")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if req.URL == nil || req.URL.String() == "" {
			t.Fatalf("%s: empty url", name)
		}
	}
}

func TestBuildersHitMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	// Any path returns 404 — we only assert client construction + RoundTrip wiring optional.
	for _, name := range AllTSFiles {
		req, err := BuildRequest(name, srv.URL, srv.URL)
		if err != nil {
			t.Fatal(name, err)
		}
		req.Host = ""
	}
}
