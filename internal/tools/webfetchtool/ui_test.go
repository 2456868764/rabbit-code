package webfetchtool

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetToolUseSummary(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://docs.anthropic.com/en/docs", "docs.anthropic.com"},
		{"http://example.com/path?q=1", "example.com"},
		{"not-a-url", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := GetToolUseSummary(c.url); got != c.want {
			t.Errorf("GetToolUseSummary(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestToolUseDescription(t *testing.T) {
	if d := ToolUseDescription("https://example.com/p"); d != "Claude wants to fetch content from example.com" {
		t.Fatalf("got %q", d)
	}
	if d := ToolUseDescription("not-url"); d != "Claude wants to fetch content from this URL" {
		t.Fatalf("fallback: %q", d)
	}
}

func TestActivityDescription(t *testing.T) {
	if d := ActivityDescription("https://example.com"); d != "Fetching example.com" {
		t.Fatalf("got %q", d)
	}
	if d := ActivityDescription(""); d != "Fetching web page" {
		t.Fatalf("fallback: %q", d)
	}
}

func TestToAutoClassifierInput(t *testing.T) {
	if s := ToAutoClassifierInput("https://example.com", "summarize"); s != "https://example.com: summarize" {
		t.Fatalf("with prompt: %q", s)
	}
	if s := ToAutoClassifierInput("https://example.com", ""); s != "https://example.com" {
		t.Fatalf("no prompt: %q", s)
	}
}

func TestValidateInput_valid(t *testing.T) {
	r := ValidateInput("https://example.com")
	if !r.Result {
		t.Fatalf("want valid, got %+v", r)
	}
}

func TestValidateInput_invalid(t *testing.T) {
	r := ValidateInput("://bad")
	if r.Result {
		t.Fatal("want invalid")
	}
	if r.ErrorCode != 1 {
		t.Fatalf("errorCode %d", r.ErrorCode)
	}
	if r.Meta["reason"] != "invalid_url" {
		t.Fatalf("meta %v", r.Meta)
	}
}

func TestOutputTypeExported(t *testing.T) {
	out := Output{
		Bytes:      100,
		Code:       200,
		CodeText:   "OK",
		Result:     "text",
		DurationMs: 50,
		URL:        "https://example.com",
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["result"] != "text" {
		t.Fatalf("output result field: %v", m)
	}
}

func TestRun_strictJSON_rejectsUnknownFields(t *testing.T) {
	srv, client := tlsServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})
	defer srv.Close()

	rc := &RunContext{HTTPClient: client}
	ctx := WithRunContext(context.Background(), rc)
	_, err := New().Run(ctx, []byte(`{"url":"`+srv.URL+`","prompt":"p","unknown_field":true}`))
	if err == nil {
		t.Fatal("want error for unknown field")
	}
}
