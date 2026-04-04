package compact

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestFindLastCompactBoundaryIndexTranscriptJSON(t *testing.T) {
	raw := []byte(`[{"role":"user","content":[{"type":"text","text":"a"}]},{"type":"system","subtype":"compact_boundary","content":"x","compactMetadata":{"trigger":"auto","preTokens":1}}]`)
	idx, err := FindLastCompactBoundaryIndexTranscriptJSON(raw)
	if err != nil || idx != 1 {
		t.Fatalf("idx=%d err=%v", idx, err)
	}
	idx2, err := FindLastCompactBoundaryIndexTranscriptJSON([]byte(`[]`))
	if err != nil || idx2 != -1 {
		t.Fatalf("empty: idx=%d", idx2)
	}
}

func TestGetMessagesAfterCompactBoundaryTranscriptJSON(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":[{"type":"text","text":"old"}]},
		{"type":"system","subtype":"compact_boundary","content":"Conversation compacted","compactMetadata":{"trigger":"manual","preTokens":1}},
		{"role":"user","content":[{"type":"text","text":"new"}]}
	]`)
	out, err := GetMessagesAfterCompactBoundaryTranscriptJSON(raw, AfterCompactBoundaryOptions{IncludeSnipped: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"new"`) || strings.Contains(string(out), `"old"`) {
		t.Fatalf("%s", out)
	}
}

func TestGetMessagesAfterCompactBoundaryTranscriptJSON_snipFilter(t *testing.T) {
	t.Setenv(features.EnvHistorySnip, "1")
	raw := []byte(`[
		{"type":"system","subtype":"compact_boundary","content":"x","compactMetadata":{"trigger":"auto","preTokens":1}},
		{"role":"user","snipped":true,"content":[{"type":"text","text":"gone"}]},
		{"role":"user","content":[{"type":"text","text":"keep"}]}
	]`)
	out, err := GetMessagesAfterCompactBoundaryTranscriptJSON(raw, AfterCompactBoundaryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "gone") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(string(out), "keep") {
		t.Fatalf("%s", out)
	}
}

func TestPartialCompactPartitionTranscriptJSON(t *testing.T) {
	raw := []byte(`[
		{"type":"user","message":{"content":[{"type":"text","text":"a"}]}},
		{"type":"system","subtype":"compact_boundary","content":"x"},
		{"type":"user","isCompactSummary":true,"message":{"content":[{"type":"text","text":"sum"}]}},
		{"type":"user","message":{"content":[{"type":"text","text":"tail"}]}}
	]`)
	sum, keep, err := PartialCompactPartitionTranscriptJSON(raw, 1, PartialCompactUpTo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sum), `"a"`) || strings.Contains(string(sum), "tail") {
		t.Fatalf("sum=%s", sum)
	}
	if !strings.Contains(string(keep), "tail") || strings.Contains(string(keep), "compact_boundary") {
		t.Fatalf("keep=%s", keep)
	}
}

func TestSelectPartialCompactAPIMessagesTranscriptJSON(t *testing.T) {
	raw := []byte(`[{"role":"user","content":[{"type":"text","text":"a"}]},{"role":"assistant","content":[{"type":"text","text":"b"}]}]`)
	full, err := SelectPartialCompactAPIMessagesTranscriptJSON(raw, 1, PartialCompactFrom)
	if err != nil || string(full) != string(raw) {
		t.Fatalf("from: %v %s", err, full)
	}
	part, err := SelectPartialCompactAPIMessagesTranscriptJSON(raw, 1, PartialCompactUpTo)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(part), `"b"`) {
		t.Fatalf("up_to should be first msg only: %s", part)
	}
}

func TestCreateCompactBoundaryMessageJSON(t *testing.T) {
	b, err := CreateCompactBoundaryMessageJSON("manual", 42, "parent-uuid", "ctx", 7)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["subtype"] != "compact_boundary" {
		t.Fatal(m)
	}
	cm, _ := m["compactMetadata"].(map[string]interface{})
	if cm["trigger"] != "manual" || int(cm["preTokens"].(float64)) != 42 {
		t.Fatalf("%v", cm)
	}
	if m["logicalParentUuid"] != "parent-uuid" {
		t.Fatal(m)
	}
}

func TestAttachPreCompactDiscoveredToolsToBoundaryJSON(t *testing.T) {
	b, err := CreateCompactBoundaryMessageJSON("auto", 1, "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	out, err := AttachPreCompactDiscoveredToolsToBoundaryJSON(b, []string{"z", "a", "a"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	cm, _ := m["compactMetadata"].(map[string]interface{})
	arr, _ := cm["preCompactDiscoveredTools"].([]interface{})
	if len(arr) != 2 {
		t.Fatalf("%v", arr)
	}
	if arr[0].(string) != "a" || arr[1].(string) != "z" {
		t.Fatalf("%v", arr)
	}
}

func TestExtractDiscoveredToolNamesFromTranscriptJSON(t *testing.T) {
	raw := []byte(`[
		{"type":"system","subtype":"compact_boundary","content":"x","compactMetadata":{"preCompactDiscoveredTools":["carried"]}},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":[
			{"type":"tool_reference","tool_name":"live"}
		]}]}
	]`)
	names, err := ExtractDiscoveredToolNamesFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "carried" || names[1] != "live" {
		t.Fatalf("%v", names)
	}
}

func TestBuildCompactStreamRequestMessagesJSON(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":[{"type":"text","text":"drop"}]},
		{"type":"system","subtype":"compact_boundary","content":"c","compactMetadata":{"trigger":"auto","preTokens":1}},
		{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"QQ=="}}]}
	]`)
	out, err := BuildCompactStreamRequestMessagesJSON(raw, AfterCompactBoundaryOptions{IncludeSnipped: true}, "SUMMARY_PROMPT")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "drop") {
		t.Fatalf("should start after boundary: %s", out)
	}
	if !strings.Contains(string(out), "SUMMARY_PROMPT") || !strings.Contains(string(out), "[image]") {
		t.Fatalf("%s", out)
	}
}

func TestCompactSummaryLooksLikePromptTooLong(t *testing.T) {
	if !CompactSummaryLooksLikePromptTooLong("Prompt is too long: 1 > 0") {
		t.Fatal()
	}
	if CompactSummaryLooksLikePromptTooLong("ok summary") {
		t.Fatal()
	}
}

func TestBuildDefaultPostCompactTranscriptJSON(t *testing.T) {
	transcript := []byte(`[
		{"uuid":"u1","role":"user","content":[{"type":"text","text":"hi"}]},
		{"role":"assistant","content":[{"type":"text","text":"yo"}]}
	]`)
	rawSummary := "<summary>done</summary>"
	out, err := BuildDefaultPostCompactTranscriptJSON(transcript, rawSummary, PostCompactTranscriptOptions{
		AutoCompact: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "compact_boundary") || !strings.Contains(string(out), "done") {
		t.Fatalf("%s", out)
	}
}

func TestBuildDefaultPostCompactTranscriptJSON_extraAttachments(t *testing.T) {
	att, err := CreateAttachmentMessageJSON(map[string]interface{}{"type": "plan_mode", "planExists": false})
	if err != nil {
		t.Fatal(err)
	}
	transcript := []byte(`[{"uuid":"u1","role":"user","content":[{"type":"text","text":"hi"}]}]`)
	out, err := BuildDefaultPostCompactTranscriptJSON(transcript, "<summary>x</summary>", PostCompactTranscriptOptions{
		ExtraAttachmentsJSON: []json.RawMessage{att},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "plan_mode") {
		t.Fatalf("%s", out)
	}
}

func TestShouldSuppressCompactErrorNotification(t *testing.T) {
	if !ShouldSuppressCompactErrorNotification(ErrorMessageUserAbort) {
		t.Fatal()
	}
	if !ShouldSuppressCompactErrorNotification(ErrorMessageNotEnoughMessages) {
		t.Fatal()
	}
	if ShouldSuppressCompactErrorNotification("other") {
		t.Fatal()
	}
}
