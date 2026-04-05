package webfetchtool

import (
	"net/http"
	"testing"
)

func TestResponseReasonPhrase(t *testing.T) {
	r := &http.Response{StatusCode: 200, Status: "200 OK"}
	if responseReasonPhrase(r) != "OK" {
		t.Fatalf("got %q", responseReasonPhrase(r))
	}
	r = &http.Response{StatusCode: 418, Status: "418 I'm a teapot"}
	if responseReasonPhrase(r) != "I'm a teapot" {
		t.Fatalf("got %q", responseReasonPhrase(r))
	}
}
