package websearchtool

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// WebSearch implements tools.Tool (WebSearchTool.ts headless execution).
type WebSearch struct{}

// New returns a WebSearch tool.
func New() *WebSearch { return &WebSearch{} }

func (w *WebSearch) Name() string { return WebSearchToolName }

func (w *WebSearch) Aliases() []string { return nil }

// Input mirrors WebSearchTool inputSchema (strictObject).
type Input struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// SearchHit mirrors searchHitSchema in WebSearchTool.ts.
type SearchHit struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// SearchResultBlock mirrors searchResultSchema in WebSearchTool.ts.
type SearchResultBlock struct {
	ToolUseID string      `json:"tool_use_id"`
	Content   []SearchHit `json:"content"`
}

// Output mirrors WebSearchTool.ts export type Output (outputSchema fields).
type Output struct {
	Query           string  `json:"query"`
	Results         []any   `json:"results"`
	DurationSeconds float64 `json:"durationSeconds"`
}

// IsEnabled mirrors isEnabled() in WebSearchTool.ts (provider/model gating).
// provider is one of "firstParty", "vertex", "foundry", "bedrock", etc.
func IsEnabled(provider, model string) bool {
	switch provider {
	case "firstParty":
		return true
	case "vertex":
		return strings.Contains(model, "claude-opus-4") ||
			strings.Contains(model, "claude-sonnet-4") ||
			strings.Contains(model, "claude-haiku-4")
	case "foundry":
		return true
	}
	return false
}

const headlessNoBackend = "Web search is not available in this headless runner. Wire websearchtool.RunContext.ExecuteSearch to perform live search (Messages API web_search_20250305)."

// Run validates input and returns JSON output (Output schema upstream).
func (w *WebSearch) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in, err := DecodeInputStrictJSON(inputJSON)
	if err != nil {
		return nil, err
	}
	if err := ValidateInput(in); err != nil {
		return nil, err
	}
	rc := RunContextFrom(ctx)

	var results []any
	var runErr error
	if rc != nil && rc.ExecuteSearch != nil {
		results, runErr = rc.ExecuteSearch(ctx, in)
		if runErr != nil {
			return nil, runErr
		}
	} else {
		results = []any{headlessNoBackend}
	}

	out := Output{
		Query:           in.Query,
		Results:         results,
		DurationSeconds: time.Since(start).Seconds(),
	}
	return json.Marshal(out)
}
