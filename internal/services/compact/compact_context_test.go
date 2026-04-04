package compact

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestExecutorSuggestMeta_roundTrip(t *testing.T) {
	ctx := context.Background()
	m := ExecutorSuggestMeta{AutoCompact: true, ReactiveCompact: false}
	ctx2 := ContextWithExecutorSuggestMeta(ctx, m)
	got, ok := ExecutorSuggestMetaFromContext(ctx2)
	if !ok || got.AutoCompact != true || got.ReactiveCompact != false {
		t.Fatalf("%+v %v", got, ok)
	}
	_, ok2 := ExecutorSuggestMetaFromContext(ctx)
	if ok2 {
		t.Fatal("expected no meta on bare ctx")
	}
}

func TestDefaultCompactStreamingToolsJSON_readOnly(t *testing.T) {
	raw, err := DefaultCompactStreamingToolsJSON(false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"Read"`) {
		t.Fatalf("%s", raw)
	}
	if strings.Contains(string(raw), ToolSearchToolName) {
		t.Fatal("tool search off")
	}
}

func TestDefaultCompactStreamingToolsJSON_withSearch(t *testing.T) {
	raw, err := DefaultCompactStreamingToolsJSON(true)
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) != 2 {
		t.Fatalf("%v %s", err, raw)
	}
}
