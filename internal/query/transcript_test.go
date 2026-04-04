package query

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTemplateMarkdownAppendix(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "foo.md"), []byte("# Hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadTemplateMarkdownAppendix(dir, []string{"foo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "## Template foo") || !strings.Contains(s, "# Hi") {
		t.Fatalf("%q", s)
	}
}

func TestLoadTemplateMarkdownAppendix_rejectsPathInName(t *testing.T) {
	_, err := LoadTemplateMarkdownAppendix(t.TempDir(), []string{"../x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyUserTextHints(t *testing.T) {
	out := ApplyUserTextHints("hello", UserTextHintFlags{Ultrathink: true})
	if out == "" || out == "hello" {
		t.Fatal(out)
	}
	out2 := ApplyUserTextHints("x", UserTextHintFlags{ContextCollapse: true, Ultraplan: true})
	if out2 == "" || out2 == "x" {
		t.Fatal(out2)
	}
	out3 := ApplyUserTextHints("z", UserTextHintFlags{SessionRestore: true})
	if out3 == "" || out3 == "z" {
		t.Fatal(out3)
	}
}

func TestFormatHeadlessModeTags_order(t *testing.T) {
	got := FormatHeadlessModeTags(UserTextHintFlags{
		Ultraplan: true, Ultrathink: true, ContextCollapse: true, SessionRestore: true,
	})
	want := "context_collapse,ultrathink,ultraplan,session_restore"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTrimTranscriptPrefixWhileOverBudget(t *testing.T) {
	raw, err := InitialUserMessagesJSON("u")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = AppendAssistantTextMessage(raw, strings.Repeat("a", 500))
	if err != nil {
		t.Fatal(err)
	}
	out, rounds, err := TrimTranscriptPrefixWhileOverBudget(raw, 200, 3)
	if err != nil {
		t.Fatal(err)
	}
	if rounds < 1 {
		t.Fatalf("rounds=%d", rounds)
	}
	if len(out) > len(raw) {
		t.Fatal("expected shrink")
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil || len(arr) == 0 {
		t.Fatalf("bad out %s", out)
	}
}

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
