package compact

import (
	"encoding/json"
	"testing"
	"time"
)

// microcompactAPIStateMarker mirrors querydeps.MicrocompactAPIStateMarker (avoid compact_test → querydeps → compact cycle).
type microcompactAPIStateMarker interface {
	MarkToolsSentToAPIState()
}

func TestMicrocompactEditBuffer_implementsMicrocompactMarker(t *testing.T) {
	var m microcompactAPIStateMarker = &MicrocompactEditBuffer{}
	m.MarkToolsSentToAPIState()
}

func TestMicrocompactEditBuffer_flow(t *testing.T) {
	var b MicrocompactEditBuffer
	b.SetPendingCacheEdits(json.RawMessage(`{"x":1}`))
	got := b.ConsumePendingCacheEdits()
	if string(got) != `{"x":1}` {
		t.Fatalf("pending %s", got)
	}
	if b.ConsumePendingCacheEdits() != nil {
		t.Fatal("consume clears")
	}
	b.PinCacheEdits(2, json.RawMessage(`{"pin":true}`))
	p := b.GetPinnedCacheEdits()
	if len(p) != 1 || p[0].UserMessageIndex != 2 {
		t.Fatalf("%+v", p)
	}
	b.MarkToolsSentToAPIState()
	if !b.ToolsSentToAPI() {
		t.Fatal("tools sent")
	}
	b.ResetMicrocompactState()
	if len(b.GetPinnedCacheEdits()) != 0 {
		t.Fatal("reset pinned")
	}
	if b.ToolsSentToAPI() {
		t.Fatal("reset flag")
	}
}

func TestMicrocompactMessagesAPIJSON_clearsCompactWarningSuppression(t *testing.T) {
	SuppressCompactWarning()
	if !CompactWarningSuppressed() {
		t.Fatal("expected suppressed")
	}
	raw := []byte(`[{"role":"user","content":[{"type":"text","text":"hi"}]}]`)
	out, _, _, _, err := MicrocompactMessagesAPIJSON(raw, "repl_main_thread", time.Now(), time.Now(), "claude-3-5-sonnet-20241022", nil)
	if err != nil {
		t.Fatal(err)
	}
	if CompactWarningSuppressed() {
		t.Fatal("microcompactMessages starts with clearCompactWarningSuppression")
	}
	if string(out) != string(raw) {
		t.Fatalf("unchanged transcript: %s", out)
	}
}

func TestMicrocompactMessagesAPIJSON_timeBasedMatchesRunMaybe(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_GAP_MINUTES", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_KEEP_RECENT", "1")
	raw := []byte(`[
	  {"role":"assistant","content":[
	    {"type":"tool_use","id":"tool_a","name":"Read","input":{}},
	    {"type":"tool_use","id":"tool_b","name":"Read","input":{}}
	  ]},
	  {"role":"user","content":[
	    {"type":"tool_result","tool_use_id":"tool_a","content":"aaaaaaaaaaaaaaaa"},
	    {"type":"tool_result","tool_use_id":"tool_b","content":"bbbbbbbbbbbbbbbb"}
	  ]}
	]`)
	lastAssist := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	wantOut, wantTok, wantCh, err := RunMaybeTimeBasedMicrocompactAPIJSON(raw, "repl_main_thread", now, lastAssist, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotOut, gotTok, gotCh, _, err := MicrocompactMessagesAPIJSON(raw, "repl_main_thread", now, lastAssist, "claude-3-5-sonnet-20241022", nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotCh != wantCh || gotTok != wantTok || string(gotOut) != string(wantOut) {
		t.Fatalf("got ch=%v tok=%v out=%s want ch=%v tok=%v", gotCh, gotTok, gotOut, wantCh, wantTok)
	}
}

func TestEstimateMessageTokensFromAPIMessagesJSON_textAndToolUse(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":[{"type":"text","text":"abcd"}]},
		{"role":"assistant","content":[{"type":"tool_use","id":"x","name":"Read","input":{"p":1}}]}
	]`)
	n, err := EstimateMessageTokensFromAPIMessagesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	base := mcBytesAsTokens("abcd") + mcBytesAsTokens(`Read{"p":1}`)
	want := (base*4 + 2) / 3
	if n != want {
		t.Fatalf("got %d want %d", n, want)
	}
}

func TestEstimateMessageTokensFromAPIMessagesJSON_largeBase64Image(t *testing.T) {
	longB64 := make([]byte, 4000)
	for i := range longB64 {
		longB64[i] = 'A'
	}
	raw := []byte(`[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"` + string(longB64) + `"}}]}]`)
	n, err := EstimateMessageTokensFromAPIMessagesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n <= ImageDocumentTokenEstimate {
		t.Fatalf("expected large base64 to raise estimate, got %d", n)
	}
}

func TestIsMainThreadQuerySource(t *testing.T) {
	if !IsMainThreadQuerySource("") || !IsMainThreadQuerySource("repl_main_thread") ||
		!IsMainThreadQuerySource("repl_main_thread:outputStyle:foo") {
		t.Fatal("expected main-thread sources")
	}
	if IsMainThreadQuerySource("session_memory") {
		t.Fatal("fork should not be main thread")
	}
}

func TestIsMainThreadPostCompactSource_includesSDK(t *testing.T) {
	if !IsMainThreadPostCompactSource("sdk") {
		t.Fatal("sdk should be main-thread for post-compact cleanup")
	}
	if IsMainThreadPostCompactSource("agent:foo") {
		t.Fatal("subagent should not run main-thread resets")
	}
}

func TestCollectCompactableToolUseIDsFromTranscriptJSON(t *testing.T) {
	raw := []byte(`[
	  {"role":"user","content":[{"type":"text","text":"hi"}]},
	  {"role":"assistant","content":[
	    {"type":"tool_use","id":"a1","name":"Read","input":{}},
	    {"type":"tool_use","id":"b1","name":"NotebookEdit","input":{}}
	  ]}
	]`)
	ids, err := CollectCompactableToolUseIDsFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "a1" {
		t.Fatalf("got %v", ids)
	}
}
