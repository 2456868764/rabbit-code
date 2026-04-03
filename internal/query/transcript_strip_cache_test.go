package query

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestStripCacheControlFromMessagesJSON_removesNested(t *testing.T) {
	raw := json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral"}}]}]`)
	out, changed, err := StripCacheControlFromMessagesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	if bytes.Contains(out, []byte("cache_control")) {
		t.Fatalf("still has cache_control: %s", out)
	}
	var msgs []map[string]interface{}
	if err := json.Unmarshal(out, &msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatal(len(msgs))
	}
}

func TestStripCacheControlFromMessagesJSON_noopWhenAbsent(t *testing.T) {
	raw := json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"hi"}]}]`)
	out, changed, err := StripCacheControlFromMessagesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected unchanged")
	}
	// Re-encode can reorder keys; semantic equality is what matters.
	var a, b []interface{}
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &b); err != nil {
		t.Fatal(err)
	}
	ae, _ := json.Marshal(a)
	be, _ := json.Marshal(b)
	if !bytes.Equal(ae, be) {
		t.Fatalf("semantic differ: %s vs %s", out, raw)
	}
}

func TestStripCacheControlFromMessagesJSON_emptyErrors(t *testing.T) {
	_, _, err := StripCacheControlFromMessagesJSON(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	_, _, err = StripCacheControlFromMessagesJSON(json.RawMessage(`   `))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStripCacheControlFromMessagesJSON_invalidJSON(t *testing.T) {
	_, _, err := StripCacheControlFromMessagesJSON(json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStripCacheControlFromMessagesJSON_nestedInToolInput(t *testing.T) {
	raw := json.RawMessage(`[{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"x","input":{"a":1},"cache_control":{"type":"ephemeral"}}]}]`)
	out, changed, err := StripCacheControlFromMessagesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected cache_control removed from tool_use block")
	}
	if bytes.Contains(out, []byte("cache_control")) {
		t.Fatalf("still has cache_control: %s", out)
	}
}
