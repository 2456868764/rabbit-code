package websearchtool

import "testing"

func TestExtractQueryFromPartialWebSearchInputJSON(t *testing.T) {
	q, ok := ExtractQueryFromPartialWebSearchInputJSON(`{"query":"hello"}`)
	if !ok || q != "hello" {
		t.Fatalf("got %q ok=%v", q, ok)
	}
	q, ok = ExtractQueryFromPartialWebSearchInputJSON(`prefix {"query":"a \"quote\""}`)
	if !ok || q != `a "quote"` {
		t.Fatalf("escaped %q ok=%v", q, ok)
	}
	if _, ok := ExtractQueryFromPartialWebSearchInputJSON(`{"query":`); ok {
		t.Fatal("incomplete should fail")
	}
}
