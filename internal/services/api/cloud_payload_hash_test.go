package anthropic

import (
	"net/http"
	"strings"
	"testing"
)

func TestHashRequestPayloadSHA256Hex_restoresBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	h, err := hashRequestPayloadSHA256Hex(req)
	if err != nil {
		t.Fatal(err)
	}
	want := "015abd7f5cc57a2dd94b7590f04ad8084273905ee33ec5cebeae62276a97f862" // sha256 of {"a":1}
	if h != want {
		t.Fatalf("hash=%s want %s", h, want)
	}
	if req.GetBody == nil {
		t.Fatal("GetBody nil")
	}
	rc, err := req.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	_ = rc.Close()
}
