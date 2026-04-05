package toolsearchtool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestToolSearch_disabledWhenStandardMode(t *testing.T) {
	t.Setenv("RABBIT_CODE_TOOL_SEARCH_OPTIMISTIC", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "false")
	if features.GetToolSearchMode() != features.ToolSearchModeStandard {
		t.Fatalf("mode %v", features.GetToolSearchMode())
	}
	if features.ToolSearchEnabledOptimistic() {
		t.Fatal("expected optimistic off")
	}
	_, err := New().Run(context.Background(), []byte(`{"query":"x"}`))
	if err == nil {
		t.Fatal("expected error when tool search optimistic off")
	}
}

func TestToolSearch_selectRead(t *testing.T) {
	t.Setenv("RABBIT_CODE_TOOL_SEARCH_OPTIMISTIC", "1")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	out, err := New().Run(context.Background(), []byte(`{"query":"select:Read"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	matches, _ := m["matches"].([]any)
	if len(matches) != 1 || matches[0] != "Read" {
		t.Fatalf("%v", m)
	}
}

func TestToolSearch_keywordNotebook(t *testing.T) {
	t.Setenv("RABBIT_CODE_TOOL_SEARCH_OPTIMISTIC", "1")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	out, err := New().Run(context.Background(), []byte(`{"query":"notebook","max_results":3}`))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Matches []string `json:"matches"`
		Total   int      `json:"total_deferred_tools"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m.Total < 1 {
		t.Fatalf("total %d", m.Total)
	}
	if len(m.Matches) < 1 || m.Matches[0] != "NotebookEdit" {
		t.Fatalf("%+v", m)
	}
}

func TestMapToolSearch_noMatches(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"matches":                 []string{},
		"query":                   "zzz",
		"total_deferred_tools":    2,
		"pending_mcp_servers":     []string{"srv"},
	})
	v := MapToolSearchToolResultForMessagesAPI(raw)
	s, ok := v.(string)
	if !ok || s == "" {
		t.Fatalf("%T %v", v, v)
	}
}

func TestMapToolSearch_references(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"matches":              []string{"WebFetch"},
		"query":                "x",
		"total_deferred_tools": 4,
	})
	v := MapToolSearchToolResultForMessagesAPI(raw)
	arr, ok := v.([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("%T %+v", v, v)
	}
	ref, ok := arr[0].(map[string]any)
	if !ok || ref["tool_name"] != "WebFetch" {
		t.Fatalf("%+v", ref)
	}
}
