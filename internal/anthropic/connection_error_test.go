package anthropic

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
)

type roundTripErrFunc func(*http.Request) (*http.Response, error)

func (f roundTripErrFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClassifyRoundTripError_OpError(t *testing.T) {
	inner := errors.New("connection refused")
	err := &net.OpError{Op: "dial", Net: "tcp", Err: inner}
	out := ClassifyRoundTripError(err)
	api, ok := out.(*APIError)
	if !ok || api.Kind != KindConnection {
		t.Fatalf("got %v", out)
	}
}

func TestClassifyRoundTripError_Context(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ctx.Err()
	out := ClassifyRoundTripError(err)
	if out != err {
		t.Fatal(out)
	}
}

func TestClassifyRoundTripError_Passthrough(t *testing.T) {
	err := errors.New("custom")
	if ClassifyRoundTripError(err) != err {
		t.Fatal()
	}
}

func TestDoRequest_LastRoundTripErrorClassified(t *testing.T) {
	var n int
	rt := roundTripErrFunc(func(*http.Request) (*http.Response, error) {
		n++
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")}
	})
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:1", nil)
	_, err := DoRequest(context.Background(), rt, req, Policy{MaxAttempts: 1, Retry529429: false})
	api, ok := err.(*APIError)
	if !ok || api.Kind != KindConnection {
		t.Fatalf("got %v", err)
	}
	if n != 1 {
		t.Fatalf("attempts=%d", n)
	}
}
