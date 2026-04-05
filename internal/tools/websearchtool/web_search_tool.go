package websearchtool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
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

type output struct {
	Query           string  `json:"query"`
	Results         []any   `json:"results"`
	DurationSeconds float64 `json:"durationSeconds"`
}

const headlessNoBackend = "Web search is not available in this headless runner. Wire websearchtool.RunContext.ExecuteSearch to perform live search (Messages API web_search_20250305)."

// ValidateInput mirrors WebSearchTool.validateInput plus z.string().min(2) on query.
func ValidateInput(in Input) error {
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return fmt.Errorf("Error: Missing query")
	}
	if utf8.RuneCountInString(q) < 2 {
		return fmt.Errorf("websearchtool: query must be at least 2 characters")
	}
	if len(in.AllowedDomains) > 0 && len(in.BlockedDomains) > 0 {
		return fmt.Errorf("Error: Cannot specify both allowed_domains and blocked_domains in the same request")
	}
	return nil
}

// Run validates input and returns JSON output (Output schema upstream).
func (w *WebSearch) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var in Input
	if err := json.Unmarshal(inputJSON, &in); err != nil {
		return nil, fmt.Errorf("websearchtool: invalid json: %w", err)
	}
	if err := ValidateInput(in); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(in.Query)
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

	out := output{
		Query:           query,
		Results:         results,
		DurationSeconds: time.Since(start).Seconds(),
	}
	return json.Marshal(out)
}
