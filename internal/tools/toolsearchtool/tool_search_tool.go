package toolsearchtool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// ToolSearch implements tools.Tool for ToolSearchTool.ts.
type ToolSearch struct{}

// New returns a ToolSearch tool.
func New() *ToolSearch { return &ToolSearch{} }

func (t *ToolSearch) Name() string { return ToolSearchToolName }

func (t *ToolSearch) Aliases() []string { return nil }

type toolSearchInput struct {
	Query      string `json:"query"`
	MaxResults *int   `json:"max_results,omitempty"`
}

func effectiveCatalog(rc *RunContext) []ToolEntry {
	if rc != nil && len(rc.FullCatalog) > 0 {
		return rc.FullCatalog
	}
	return DefaultCatalog()
}

func effectiveDeferredNames(rc *RunContext) []string {
	if rc != nil && len(rc.DeferredToolNames) > 0 {
		return rc.DeferredToolNames
	}
	return DefaultDeferredToolNames()
}

func pendingFromCtx(rc *RunContext) []string {
	if rc == nil {
		return nil
	}
	return rc.PendingMCPServers
}

// Run implements tools.Tool.
func (t *ToolSearch) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if !features.ToolSearchEnabledOptimistic() {
		return nil, errors.New("toolsearchtool: ToolSearch is disabled (ENABLE_TOOL_SEARCH / tool search mode)")
	}
	var in toolSearchInput
	if err := json.Unmarshal(inputJSON, &in); err != nil {
		return nil, fmt.Errorf("toolsearchtool: invalid json: %w", err)
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, errors.New("toolsearchtool: missing query")
	}
	max := 5
	if in.MaxResults != nil && *in.MaxResults > 0 {
		max = *in.MaxResults
	}

	rc := RunContextFrom(ctx)
	all := effectiveCatalog(rc)
	deferred := filterDeferred(all, effectiveDeferredNames(rc))

	var matches []string
	if names, ok := parseSelectQuery(q); ok {
		found, _ := resolveSelect(names, deferred, all)
		matches = found
	} else {
		matches = searchToolsWithKeywords(q, deferred, all, max)
	}

	out := map[string]any{
		"matches":               matches,
		"query":                 q,
		"total_deferred_tools": len(deferred),
	}
	if pend := pendingFromCtx(rc); len(pend) > 0 && len(matches) == 0 {
		out["pending_mcp_servers"] = pend
	}
	return json.Marshal(out)
}
