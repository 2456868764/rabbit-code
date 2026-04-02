package anthropic

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type headerSigner struct {
	key, val string
}

func (s headerSigner) Sign(_ context.Context, req *http.Request) error {
	req.Header.Set(s.key, s.val)
	return nil
}

func TestSigningTransport_invokesSignerBeforeBase(t *testing.T) {
	var saw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saw = r.Header.Get("X-Test-Signed")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	base := http.DefaultTransport
	rt := &SigningTransport{
		Base:   base,
		Signer: headerSigner{key: "X-Test-Signed", val: "1"},
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if saw != "1" {
		t.Fatalf("header=%q", saw)
	}
}

func TestSigningTransport_nilSignerDelegates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	rt := &SigningTransport{Base: http.DefaultTransport, Signer: nil}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}

func TestStubBedrockSigner(t *testing.T) {
	var s StubBedrockSigner
	err := s.Sign(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil))
	if !errors.Is(err, ErrCloudSigningNotImplemented) {
		t.Fatalf("got %v", err)
	}
}
