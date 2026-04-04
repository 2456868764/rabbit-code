package compact

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMaybeTimeBasedMicrocompactJSON_disabled(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "")
	raw := []byte(`[{"type":"assistant","timestamp":"2020-01-01T00:00:00Z","message":{"content":[]}}]`)
	out, tok, ch, err := MaybeTimeBasedMicrocompactJSON(raw, "repl_main_thread", time.Now())
	if err != nil || ch || tok != 0 || string(out) != string(raw) {
		t.Fatalf("%v %v %v %s", err, ch, tok, out)
	}
}

func TestMaybeTimeBasedMicrocompactJSON_clearsOlderToolResults(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_GAP_MINUTES", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_KEEP_RECENT", "1")
	raw := []byte(`[
	  {"type":"assistant","timestamp":"2020-01-01T00:00:00Z","message":{"content":[
	    {"type":"tool_use","id":"tool_a","name":"Read","input":{}},
	    {"type":"tool_use","id":"tool_b","name":"Read","input":{}}
	  ]}},
	  {"type":"user","message":{"content":[
	    {"type":"tool_result","tool_use_id":"tool_a","content":"aaaaaaaaaaaaaaaa"},
	    {"type":"tool_result","tool_use_id":"tool_b","content":"bbbbbbbbbbbbbbbb"}
	  ]}}
	]`)
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	out, tok, ch, err := MaybeTimeBasedMicrocompactJSON(raw, "repl_main_thread", now)
	if err != nil {
		t.Fatal(err)
	}
	if !ch || tok <= 0 {
		t.Fatalf("expected change and tokens, got ch=%v tok=%d", ch, tok)
	}
	if !strings.Contains(string(out), TimeBasedMCClearedMessage) {
		t.Fatalf("missing cleared marker: %s", out)
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	user := arr[1]
	msg := user["message"].(map[string]interface{})
	blocks := msg["content"].([]interface{})
	a := blocks[0].(map[string]interface{})
	b := blocks[1].(map[string]interface{})
	if a["content"] != TimeBasedMCClearedMessage {
		t.Fatalf("tool_a should be cleared, got %v", a["content"])
	}
	if b["content"] != "bbbbbbbbbbbbbbbb" {
		t.Fatalf("tool_b should be kept, got %v", b["content"])
	}
}

func TestRunMaybeTimeBasedMicrocompactJSON_resetsBuffer(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_GAP_MINUTES", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_KEEP_RECENT", "1")
	ClearCompactWarningSuppression()
	var buf MicrocompactEditBuffer
	buf.SetPendingCacheEdits(json.RawMessage(`{}`))
	raw := []byte(`[
	  {"type":"assistant","timestamp":"2020-01-01T00:00:00Z","message":{"content":[
	    {"type":"tool_use","id":"tool_a","name":"Read","input":{}},
	    {"type":"tool_use","id":"tool_b","name":"Read","input":{}}
	  ]}},
	  {"type":"user","message":{"content":[
	    {"type":"tool_result","tool_use_id":"tool_a","content":"aaaaaaaa"},
	    {"type":"tool_result","tool_use_id":"tool_b","content":"bbbbbbbb"}
	  ]}}
	]`)
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	_, _, ch, err := RunMaybeTimeBasedMicrocompactJSON(raw, "repl_main_thread", now, &buf)
	if err != nil || !ch {
		t.Fatalf("%v %v", err, ch)
	}
	if buf.ConsumePendingCacheEdits() != nil {
		t.Fatal("buffer should reset")
	}
	if !CompactWarningSuppressed() {
		t.Fatal("warning suppressed")
	}
}
