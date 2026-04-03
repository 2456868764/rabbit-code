package anthropic

import (
	"errors"
	"net/http"
	"sync"
	"syscall"
)

// keepAliveResetTransport wraps *http.Transport and sets DisableKeepAlive after ECONNRESET/EPIPE
// (utils/proxy.ts disableKeepAlive + withRetry.ts stale keep-alive socket path).
type keepAliveResetTransport struct {
	mu sync.Mutex
	t  *http.Transport
}

func newKeepAliveResetTransport(t *http.Transport) *keepAliveResetTransport {
	return &keepAliveResetTransport{t: t}
}

func (w *keepAliveResetTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := w.t.RoundTrip(req)
	if err != nil && isStaleConnectionReset(err) {
		w.mu.Lock()
		w.t.DisableKeepAlives = true
		w.mu.Unlock()
	}
	return resp, err
}

func isStaleConnectionReset(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE)
}
