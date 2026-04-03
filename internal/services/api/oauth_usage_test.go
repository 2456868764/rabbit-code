package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchUtilization_Mock(t *testing.T) {
	want := Utilization{
		FiveHour: &RateLimit{},
	}
	raw, _ := json.Marshal(want)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/oauth/usage" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer t" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	u, err := FetchUtilization(context.Background(), http.DefaultTransport, srv.URL, "t", true)
	if err != nil {
		t.Fatal(err)
	}
	if u.FiveHour == nil {
		t.Fatal("expected five_hour object")
	}
}

func TestFetchUtilization_SubscriberGate(t *testing.T) {
	u, err := FetchUtilization(context.Background(), http.DefaultTransport, "http://unused", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if u.FiveHour != nil || u.SevenDay != nil {
		t.Fatalf("expected empty utilization, got %+v", u)
	}
}
