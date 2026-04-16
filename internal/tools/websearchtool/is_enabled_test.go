package websearchtool

import "testing"

func TestIsEnabled(t *testing.T) {
	cases := []struct {
		provider string
		model    string
		want     bool
	}{
		// firstParty — always true
		{"firstParty", "claude-opus-4-5", true},
		{"firstParty", "claude-3-5-sonnet", true},
		{"firstParty", "", true},
		// vertex — only Claude 4.x families
		{"vertex", "claude-opus-4-20250514", true},
		{"vertex", "claude-sonnet-4-5", true},
		{"vertex", "claude-haiku-4-0", true},
		{"vertex", "claude-3-5-sonnet-20241022", false},
		{"vertex", "claude-3-opus-20240229", false},
		{"vertex", "", false},
		// foundry — always true
		{"foundry", "any-model", true},
		{"foundry", "", true},
		// bedrock / unknown — false
		{"bedrock", "claude-sonnet-4", false},
		{"", "claude-sonnet-4", false},
		{"unknown", "claude-opus-4", false},
	}
	for _, c := range cases {
		got := IsEnabled(c.provider, c.model)
		if got != c.want {
			t.Errorf("IsEnabled(%q, %q) = %v, want %v", c.provider, c.model, got, c.want)
		}
	}
}

func TestOutputTypeExported(t *testing.T) {
	out := Output{
		Query:           "test query",
		Results:         []any{"summary"},
		DurationSeconds: 1.23,
	}
	if out.Query != "test query" {
		t.Fatalf("Output.Query: %q", out.Query)
	}
	if out.DurationSeconds != 1.23 {
		t.Fatalf("Output.DurationSeconds: %v", out.DurationSeconds)
	}
}
