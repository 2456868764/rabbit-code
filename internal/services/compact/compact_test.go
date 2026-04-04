package compact

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

func TestRunPhase_String(t *testing.T) {
	if g, w := RunIdle.String(), "idle"; g != w {
		t.Fatalf("%q", g)
	}
}

func TestRunPhase_Next(t *testing.T) {
	if p := RunIdle.Next(true, false); p != RunAutoPending {
		t.Fatal(p)
	}
	if p := RunIdle.Next(false, true); p != RunReactivePending {
		t.Fatal(p)
	}
	if p := RunAutoPending.Next(false, false); p != RunExecuting {
		t.Fatal(p)
	}
	if p := RunExecuting.Next(false, false); p != RunIdle {
		t.Fatal(p)
	}
}

func TestParsePhase(t *testing.T) {
	if ParsePhase("auto_pending") != RunAutoPending {
		t.Fatal()
	}
	if ParsePhase("") != RunIdle {
		t.Fatal()
	}
}

func TestAfterSuccessfulCompactExecution(t *testing.T) {
	if g, w := AfterSuccessfulCompactExecution(RunExecuting), RunIdle; g != w {
		t.Fatalf("executing -> idle: got %v want %v", g, w)
	}
	if g, w := AfterSuccessfulCompactExecution(RunReactivePending), RunReactivePending; g != w {
		t.Fatalf("pending unchanged: got %v want %v", g, w)
	}
}

func TestExecutorPhaseAfterSchedule(t *testing.T) {
	if g, w := ExecutorPhaseAfterSchedule(RunAutoPending), RunExecuting; g != w {
		t.Fatalf("auto_pending: got %v want %v", g, w)
	}
	if g, w := ExecutorPhaseAfterSchedule(RunReactivePending), RunExecuting; g != w {
		t.Fatalf("reactive_pending: got %v want %v", g, w)
	}
	if g, w := ExecutorPhaseAfterSchedule(RunIdle), RunIdle; g != w {
		t.Fatalf("idle: got %v want %v", g, w)
	}
}

func TestResultPhaseAfterCompactExecutor(t *testing.T) {
	if g, w := ResultPhaseAfterCompactExecutor(RunExecuting, nil), RunIdle; g != w {
		t.Fatalf("success: got %v want %v", g, w)
	}
	if g, w := ResultPhaseAfterCompactExecutor(RunExecuting, errors.New("fail")), RunExecuting; g != w {
		t.Fatalf("error: got %v want %v", g, w)
	}
}

// Expected names match restored-src/src/services/compact/microCompact.ts COMPACTABLE_TOOLS,
// sourced from internal/tools/* and internal/utils/shell (mirrors src/tools + shellToolUtils).
func TestCompactableToolNames_matchMicroCompactTS(t *testing.T) {
	want := []string{
		"Read", "Bash", "PowerShell", "Grep", "Glob",
		"WebSearch", "WebFetch", "Edit", "Write",
	}
	for _, name := range want {
		if !IsCompactableToolName(name) {
			t.Errorf("expected %q to be compactable (drift vs microCompact.ts COMPACTABLE_TOOLS?)", name)
		}
	}
	if IsCompactableToolName("TodoWrite") {
		t.Fatal("TodoWrite should not be compactable")
	}
}

func TestRunPostCompactCleanup_resetsBufferWhenHooksNil(t *testing.T) {
	var b MicrocompactEditBuffer
	b.SetPendingCacheEdits(json.RawMessage(`{"x":1}`))
	RunPostCompactCleanup(context.Background(), "repl_main_thread", &b, nil)
	if b.ConsumePendingCacheEdits() != nil {
		t.Fatal("expected pending cleared after reset")
	}
}

// TestRunPostCompactCleanup_order mirrors postCompactCleanup.ts step order for main-thread vs subagent sources.
func TestRunPostCompactCleanup_order(t *testing.T) {
	t.Setenv("RABBIT_CODE_CONTEXT_COLLAPSE", "")
	t.Setenv("RABBIT_CODE_COMMIT_ATTRIBUTION", "")

	var order []string
	hooks := &PostCompactCleanupHooks{
		ResetContextCollapse:      func() { order = append(order, "collapse") },
		ClearUserContextCache:     func() { order = append(order, "userctx") },
		ResetMemoryFilesCache:     func(string) { order = append(order, "memfiles") },
		ClearSystemPromptSections: func() { order = append(order, "sysprompt") },
		ClearClassifierApprovals:  func() { order = append(order, "classifier") },
		ClearSpeculativeChecks:    func() { order = append(order, "speculative") },
		ClearBetaTracingState:     func() { order = append(order, "betatrace") },
		SweepFileContentCache:     func() { order = append(order, "sweep") },
		ClearSessionMessagesCache: func() { order = append(order, "session") },
	}

	RunPostCompactCleanup(context.Background(), "agent:sub", nil, hooks)
	wantSub := []string{"sysprompt", "classifier", "speculative", "betatrace", "session"}
	if len(order) != len(wantSub) {
		t.Fatalf("subagent: got %v want %v", order, wantSub)
	}
	for i := range wantSub {
		if order[i] != wantSub[i] {
			t.Fatalf("subagent step %d: got %v", i, order)
		}
	}

	order = order[:0]
	t.Setenv("RABBIT_CODE_CONTEXT_COLLAPSE", "1")
	RunPostCompactCleanup(context.Background(), "repl_main_thread:out", nil, hooks)
	wantMain := []string{"collapse", "userctx", "memfiles", "sysprompt", "classifier", "speculative", "betatrace", "session"}
	if len(order) != len(wantMain) {
		t.Fatalf("main: got %v want %v", order, wantMain)
	}
	for i := range wantMain {
		if order[i] != wantMain[i] {
			t.Fatalf("main step %d: got %v", i, order)
		}
	}

	order = order[:0]
	t.Setenv("RABBIT_CODE_COMMIT_ATTRIBUTION", "1")
	RunPostCompactCleanup(context.Background(), "agent:sub", nil, hooks)
	// Sweep is gated by COMMIT_ATTRIBUTION only; still not on subagent-only path before session... actually sweep runs before session for any source when feature on
	wantSweep := []string{"sysprompt", "classifier", "speculative", "betatrace", "sweep", "session"}
	if len(order) != len(wantSweep) {
		t.Fatalf("sweep: got %v want %v", order, wantSweep)
	}
}

func TestEffectiveCompactSummaryMaxTokens(t *testing.T) {
	if g, w := EffectiveCompactSummaryMaxTokens(0), CompactSummaryMaxOutputTokens; g != w {
		t.Fatalf("got %d want %d", g, w)
	}
	if g, w := EffectiveCompactSummaryMaxTokens(100), 100; g != w {
		t.Fatalf("got %d want %d", g, w)
	}
}

func TestMergeHookInstructions(t *testing.T) {
	if g := MergeHookInstructions("", "hook"); strings.TrimSpace(g) != "hook" {
		t.Fatalf("%q", g)
	}
	if g := MergeHookInstructions("user", ""); g != "user" {
		t.Fatalf("%q", g)
	}
	if g := MergeHookInstructions("a", "b"); g != "a\n\nb" {
		t.Fatalf("%q", g)
	}
}

func TestStripImagesFromAPIMessagesJSON(t *testing.T) {
	raw := []byte(`[{"role":"assistant","content":[{"type":"text","text":"hi"}]},{"role":"user","content":[{"type":"image","source":{"type":"base64","data":"eA=="}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"x","content":[{"type":"document","source":{"type":"base64","data":"eA=="}}]}]}]`)
	out, err := StripImagesFromAPIMessagesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `"type":"image"`) || strings.Contains(string(out), `"type":"document"`) {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(string(out), `[image]`) || !strings.Contains(string(out), `[document]`) {
		t.Fatalf("%s", out)
	}
}

func TestStripReinjectedAttachmentsFromTranscriptJSON(t *testing.T) {
	raw := []byte(`[{"type":"text","text":"x"},{"type":"attachment","attachment":{"type":"skill_discovery","x":1}},{"type":"attachment","attachment":{"type":"other"}}]`)
	t.Setenv("RABBIT_CODE_EXPERIMENTAL_SKILL_SEARCH", "")
	out, err := StripReinjectedAttachmentsFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(raw) {
		t.Fatal("expected no-op when feature off")
	}
	t.Setenv("RABBIT_CODE_EXPERIMENTAL_SKILL_SEARCH", "1")
	out2, err := StripReinjectedAttachmentsFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out2), "skill_discovery") {
		t.Fatalf("%s", out2)
	}
	if !strings.Contains(string(out2), `"type":"other"`) {
		t.Fatalf("%s", out2)
	}
}

func TestBuildPostCompactMessagesJSON(t *testing.T) {
	b, err := BuildPostCompactMessagesJSON(json.RawMessage(`{"role":"system","content":"b"}`),
		[]json.RawMessage{json.RawMessage(`{"role":"user","content":"s"}`)},
		nil,
		[]json.RawMessage{json.RawMessage(`{"role":"user","content":"a"}`)},
		nil)
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 3 {
		t.Fatalf("len %d", len(arr))
	}
}

func TestStartsWithAPIErrorPrefix(t *testing.T) {
	if !StartsWithAPIErrorPrefix("API Error: x") {
		t.Fatal()
	}
	if !StartsWithAPIErrorPrefix("Please run /login · API Error: x") {
		t.Fatal()
	}
	if StartsWithAPIErrorPrefix("ok") {
		t.Fatal()
	}
}

func TestPromptTooLongTokenGapFromAssistantJSON(t *testing.T) {
	raw := []byte(`{"role":"assistant","content":[{"type":"text","text":"Prompt is too long: 137500 tokens > 135000 maximum"}]}`)
	gap, ok := PromptTooLongTokenGapFromAssistantJSON(raw)
	if !ok || gap != 2500 {
		t.Fatalf("got %v %v", gap, ok)
	}
}

func TestTruncateSkillContentRoughTokens(t *testing.T) {
	long := strings.Repeat("abcd", 5000) // >> default budget
	out := TruncateSkillContentRoughTokens(long, 10)
	if !strings.HasSuffix(out, SkillTruncationMarker) {
		t.Fatal(out)
	}
	if strings.Contains(out[:len(out)-len(SkillTruncationMarker)], SkillTruncationMarker) {
		t.Fatal("marker only at end")
	}
	short := "hi"
	if TruncateSkillContentRoughTokens(short, 100) != short {
		t.Fatal()
	}
}

func TestGroupRawMessagesByAPIRound_JSONParity(t *testing.T) {
	raw := []byte(`[
	  {"type":"user","message":{}},
	  {"type":"assistant","message":{"id":"r1"}},
	  {"type":"assistant","message":{"id":"r1"}},
	  {"type":"assistant","message":{"id":"r2"}}
	]`)
	var lines []json.RawMessage
	if err := json.Unmarshal(raw, &lines); err != nil {
		t.Fatal(err)
	}
	g := GroupRawMessagesByAPIRound(lines)
	if len(g) != 3 {
		t.Fatalf("want 3 groups got %d", len(g))
	}
}

func TestTruncateHeadForPTLRetryTranscriptJSON_prependsMarker(t *testing.T) {
	msgs := []byte(`[
		{"role":"user","content":[{"type":"text","text":"u1"}]},
		{"role":"assistant","id":"a1","content":[{"type":"text","text":"x"}]},
		{"role":"user","content":[{"type":"text","text":"u2"}]},
		{"role":"assistant","id":"a2","content":[{"type":"text","text":"y"}]}
	]`)
	asst := []byte(`{"role":"assistant","content":[{"type":"text","text":"Prompt is too long: 999999 tokens > 1 maximum"}]}`)
	out, ok := TruncateHeadForPTLRetryTranscriptJSON(msgs, asst)
	if !ok {
		t.Fatal("expected ok")
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) < 2 {
		t.Fatalf("short %v", arr)
	}
	firstRole, _ := arr[0]["role"].(string)
	if firstRole != "user" {
		t.Fatalf("first role %q", firstRole)
	}
}

func TestTruncateHeadForPTLRetryTranscriptJSON_stripsLeadingMarker(t *testing.T) {
	msgs := []byte(`[
		{"role":"user","content":[{"type":"text","text":"` + PTLRetryMarker + `"}]},
		{"role":"user","content":[{"type":"text","text":"u1"}]},
		{"role":"assistant","id":"a1","content":[{"type":"text","text":"x"}]},
		{"role":"user","content":[{"type":"text","text":"u2"}]},
		{"role":"assistant","id":"a2","content":[{"type":"text","text":"y"}]}
	]`)
	asst := []byte(`{"errorDetails":"Prompt is too long: 100 tokens > 50 maximum"}`)
	out, ok := TruncateHeadForPTLRetryTranscriptJSON(msgs, asst)
	if !ok {
		t.Fatal("expected ok")
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) == 0 {
		t.Fatal()
	}
}

func TestCollectReadToolFilePathsFromTranscriptJSON(t *testing.T) {
	raw := []byte(`[
		{"role":"assistant","content":[{"type":"tool_use","id":"r1","name":"Read","input":{"file_path":"/tmp/a.go"}}]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"r1","content":"` + filereadtool.FileUnchangedStub + `"}]},
		{"role":"assistant","content":[{"type":"tool_use","id":"r2","name":"Read","input":{"file_path":"/tmp/b.go"}}]}
	]`)
	paths, err := CollectReadToolFilePathsFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := paths["/tmp/a.go"]; ok {
		t.Fatal("stubbed read should be skipped")
	}
	if _, ok := paths[filepath.Clean("/tmp/b.go")]; !ok {
		t.Fatalf("got %v", paths)
	}
}

func TestToolInputFilePathFromJSON(t *testing.T) {
	want := filepath.Clean(filepath.FromSlash("/x/y"))
	got := filepath.Clean(ToolInputFilePathFromJSON([]byte(`{"file_path":"/x/y"}`)))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if ToolInputFilePathFromJSON([]byte("")) != "" {
		t.Fatal()
	}
}

func TestAnnotateBoundaryWithPreservedSegmentJSON(t *testing.T) {
	boundary := json.RawMessage(`{"type":"system","compactMetadata":{"x":1}}`)
	out, err := AnnotateBoundaryWithPreservedSegmentJSON(boundary, "anchor", "head", "tail")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	cm, _ := m["compactMetadata"].(map[string]interface{})
	if cm == nil || cm["x"] == nil {
		t.Fatal("lost existing compactMetadata")
	}
	ps, _ := cm["preservedSegment"].(map[string]interface{})
	if ps["headUuid"] != "head" || ps["tailUuid"] != "tail" || ps["anchorUuid"] != "anchor" {
		t.Fatalf("%v", ps)
	}
}
