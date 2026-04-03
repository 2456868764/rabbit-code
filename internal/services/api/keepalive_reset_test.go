package anthropic

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"testing"
)

func TestIsStaleConnectionReset(t *testing.T) {
	if !isStaleConnectionReset(syscall.ECONNRESET) {
		t.Fatal("direct errno")
	}
	if !isStaleConnectionReset(fmt.Errorf("wrap: %w", syscall.EPIPE)) {
		t.Fatal("wrapped EPIPE")
	}
	if isStaleConnectionReset(fmt.Errorf("other")) {
		t.Fatal("false positive")
	}
}

func TestKeepAliveResetTransport_dialECONNRESETSetsDisableKeepAlives(t *testing.T) {
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, fmt.Errorf("dial: %w", syscall.ECONNRESET)
		},
	}
	w := newKeepAliveResetTransport(tr)
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:9/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !isStaleConnectionReset(err) {
		t.Fatalf("want stale reset in chain, got %v", err)
	}
	if !tr.DisableKeepAlives {
		t.Fatal("expected DisableKeepAlives")
	}
}
