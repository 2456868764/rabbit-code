package compact

import "testing"

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
