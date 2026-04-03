package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2456868764/rabbit-code/internal/services/api"
)

func TestRoundTripProbe_Mock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	resp, err := RoundTripProbe(context.Background(), http.DefaultTransport, Claude, srv.URL, srv.URL, anthropic.Policy{
		MaxAttempts:   2,
		Retry529429: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatal(resp.Status)
	}
}
