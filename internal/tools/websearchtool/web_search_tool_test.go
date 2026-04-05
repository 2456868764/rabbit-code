package websearchtool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateInput(t *testing.T) {
	if err := ValidateInput(Input{Query: "ab"}); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := ValidateInput(Input{Query: "  ab  "}); err != nil {
		t.Fatalf("trimmed valid: %v", err)
	}
	if err := ValidateInput(Input{Query: ""}); err == nil || !strings.Contains(err.Error(), "Missing query") {
		t.Fatalf("empty query: %v", err)
	}
	if err := ValidateInput(Input{Query: "a"}); err == nil {
		t.Fatal("short query want error")
	}
	if err := ValidateInput(Input{
		Query:          "ok",
		AllowedDomains: []string{"a.com"},
		BlockedDomains: []string{"b.com"},
	}); err == nil || !strings.Contains(err.Error(), "both allowed_domains and blocked_domains") {
		t.Fatalf("both domains: %v", err)
	}
}

func TestRunHeadless(t *testing.T) {
	w := New()
	out, err := w.Run(context.Background(), []byte(`{"query":"hello world"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got output
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.Query != "hello world" {
		t.Fatalf("query: %q", got.Query)
	}
	if len(got.Results) != 1 {
		t.Fatalf("results len: %d", len(got.Results))
	}
	if got.DurationSeconds < 0 {
		t.Fatalf("duration: %v", got.DurationSeconds)
	}
	s, _ := got.Results[0].(string)
	if s == "" || !strings.Contains(s, "headless") {
		t.Fatalf("result: %q", s)
	}
}

func TestRunExecuteSearch(t *testing.T) {
	ctx := WithRunContext(context.Background(), &RunContext{
		ExecuteSearch: func(_ context.Context, in Input) ([]any, error) {
			return []any{
				"Summary line.",
				map[string]any{
					"tool_use_id": "tu_1",
					"content": []map[string]string{
						{"title": "T", "url": "https://example.com"},
					},
				},
			}, nil
		},
	})
	w := New()
	out, err := w.Run(ctx, []byte(`{"query":"q1"}`))
	if err != nil {
		t.Fatal(err)
	}
	formatted := MapWebSearchToolResultForMessagesAPI(out)
	if !strings.Contains(formatted, `Web search results for query: "q1"`) {
		t.Fatalf("header: %q", formatted)
	}
	if !strings.Contains(formatted, "Summary line.") {
		t.Fatalf("summary: %q", formatted)
	}
	if !strings.Contains(formatted, "Links:") || !strings.Contains(formatted, "example.com") {
		t.Fatalf("links: %q", formatted)
	}
	if !strings.Contains(formatted, "REMINDER: You MUST include the sources") {
		t.Fatalf("reminder: %q", formatted)
	}
}

func TestMapSkipsNullEntries(t *testing.T) {
	raw := []byte(`{"query":"x","results":[null,"hi",{"content":[]} ]}`)
	s := MapWebSearchToolResultForMessagesAPI(raw)
	if !strings.Contains(s, "hi") {
		t.Fatal(s)
	}
	if !strings.Contains(s, "No links found") {
		t.Fatal(s)
	}
}

func TestPromptBodyOverrideDate(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2030-06-15")
	body := PromptBody()
	if !strings.Contains(body, "June 2030") {
		t.Fatalf("want June 2030 in: %s", body)
	}
}
