package compact

import (
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestRunCachedMicrocompactTranscriptJSON(t *testing.T) {
	t.Setenv(features.EnvCachedMicrocompact, "1")
	t.Setenv(features.EnvCachedMCTriggerThreshold, "1")
	t.Setenv(features.EnvCachedMCKeepRecent, "0")
	var buf MicrocompactEditBuffer
	tr := []byte(`[
		{"role":"assistant","content":[
			{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/a"}},
			{"type":"tool_use","id":"t2","name":"Read","input":{"file_path":"/b"}}
		]},
		{"role":"user","content":[
			{"type":"tool_result","tool_use_id":"t1","content":"a"},
			{"type":"tool_result","tool_use_id":"t2","content":"b"}
		]}
	]`)
	info, err := RunCachedMicrocompactTranscriptJSON(tr, "repl_main_thread", "claude-3-5-haiku-20241022", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if info == nil || info.PendingCacheEdits == nil {
		t.Fatal("expected pending edits")
	}
	p := buf.ConsumePendingCacheEdits()
	if len(p) == 0 {
		t.Fatal("buffer pending")
	}
	var pe MicrocompactPendingCacheEdits
	if err := json.Unmarshal(p, &pe); err != nil {
		t.Fatal(err)
	}
	if len(pe.DeletedToolIDs) == 0 {
		t.Fatalf("%+v", pe)
	}
}
