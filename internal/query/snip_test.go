package query

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestSnipDropFirstMessages(t *testing.T) {
	raw, err := InitialUserMessagesJSON("a")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = AppendAssistantTextMessage(raw, "b")
	if err != nil {
		t.Fatal(err)
	}
	out, err := SnipDropFirstMessages(raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 {
		t.Fatalf("len=%d", len(arr))
	}
	if arr[0]["role"] != "assistant" {
		t.Fatalf("%v", arr[0])
	}
	out2, err := SnipDropFirstMessages(raw, 99)
	if err != nil {
		t.Fatal(err)
	}
	var a2 []any
	if err := json.Unmarshal(out2, &a2); err != nil || len(a2) != 0 {
		t.Fatalf("want empty array got %s err=%v", out2, err)
	}
	if _, err := SnipDropFirstMessages(raw, -1); err != ErrSnipInvalidN {
		t.Fatalf("got %v", err)
	}
}

func TestReplaySnipRemovals_roundTrip(t *testing.T) {
	u1, err := InitialUserMessagesJSON("a")
	if err != nil {
		t.Fatal(err)
	}
	u1, err = AppendAssistantTextMessage(u1, "b")
	if err != nil {
		t.Fatal(err)
	}
	u1, err = AppendUserTextMessage(u1, "c")
	if err != nil {
		t.Fatal(err)
	}
	log := []SnipRemovalEntry{
		{ID: "e1", Kind: SnipRemovalKindHistorySnip, RemovedMessageCount: 1, BytesBefore: 100, BytesAfter: 50},
	}
	out, err := ReplaySnipRemovals(u1, log)
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("want 2 messages after dropping 1, got %d", len(arr))
	}
	out2, err := ReplaySnipRemovals(u1, []SnipRemovalEntry{
		{ID: "x", Kind: SnipRemovalKindHistorySnip, RemovedMessageCount: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out2, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 {
		t.Fatalf("want 1 message, got %d", len(arr))
	}
}

func TestMarshalSnipRemovalLogJSON(t *testing.T) {
	log := []SnipRemovalEntry{{ID: "abc", Kind: SnipRemovalKindSnipCompact, RemovedMessageCount: 3}}
	b, err := MarshalSnipRemovalLogJSON(log)
	if err != nil {
		t.Fatal(err)
	}
	back, err := UnmarshalSnipRemovalLogJSON(b)
	if err != nil || len(back) != 1 || back[0].ID != "abc" {
		t.Fatalf("got %+v err=%v", back, err)
	}
}

func TestNewSnipRemovalID_nonEmpty(t *testing.T) {
	id := NewSnipRemovalID()
	if len(id) < 8 {
		t.Fatalf("id %q", id)
	}
}

func TestReplaySnipRemovalsEx_removedIndices_middle(t *testing.T) {
	msgs, err := buildThreeMessageTranscript(t)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ReplaySnipRemovalsEx(msgs, []SnipRemovalEntry{
		{ID: "mid", Kind: SnipRemovalKindHistorySnip, RemovedIndices: []int{1}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("len %d", len(arr))
	}
}

func TestReplaySnipRemovalsEx_removedUuids_requiresMap(t *testing.T) {
	msgs, err := buildThreeMessageTranscript(t)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReplaySnipRemovals(msgs, []SnipRemovalEntry{
		{ID: "u", RemovedUUIDs: []string{"uuid-a"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSnipReplayUUIDMapRequired) {
		t.Fatalf("want ErrSnipReplayUUIDMapRequired, got %v", err)
	}
}

func TestReplaySnipRemovalsEx_removedUuids_withMap(t *testing.T) {
	msgs, err := buildThreeMessageTranscript(t)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ReplaySnipRemovalsEx(msgs, []SnipRemovalEntry{
		{ID: "u", RemovedUUIDs: []string{"mid"}},
	}, &SnipReplayOptions{UUIDToIndex: map[string]int{"mid": 1}})
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("len %d", len(arr))
	}
}

func TestMergeSnipRemovalLogs_dedupID(t *testing.T) {
	a := []SnipRemovalEntry{{ID: "x", RemovedMessageCount: 1}}
	b := []SnipRemovalEntry{{ID: "x", RemovedMessageCount: 2}, {ID: "y"}}
	got := MergeSnipRemovalLogs(a, b)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestSnipRemovalEntryJSON_removedUuids(t *testing.T) {
	const raw = `{"id":"1","kind":"history_snip","removedUuids":["a","b"]}`
	var e SnipRemovalEntry
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatal(err)
	}
	if len(e.RemovedUUIDs) != 2 || e.RemovedUUIDs[0] != "a" {
		t.Fatalf("%+v", e)
	}
}

func TestBuildUUIDToIndexFromMessagesJSON(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":[{"type":"text","text":"a"}],"rabbit_message_uuid":"u0"},
		{"role":"assistant","content":[{"type":"text","text":"b"}],"rabbit_message_uuid":"u1"}
	]`)
	m, err := BuildUUIDToIndexFromMessagesJSON(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if m["u0"] != 0 || m["u1"] != 1 {
		t.Fatalf("%+v", m)
	}
}

func TestBuildUUIDToIndexFromMessagesJSON_duplicateErrors(t *testing.T) {
	raw := []byte(`[
		{"role":"user","rabbit_message_uuid":"x"},
		{"role":"user","rabbit_message_uuid":"x"}
	]`)
	_, err := BuildUUIDToIndexFromMessagesJSON(raw, "")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestStripMessageFieldFromTranscriptJSON(t *testing.T) {
	raw := []byte(`[{"role":"user","rabbit_message_uuid":"a","content":[]}]`)
	out, err := StripMessageFieldFromTranscriptJSON(raw, RabbitMessageUUIDKey)
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if _, ok := arr[0][RabbitMessageUUIDKey]; ok {
		t.Fatal("field should be stripped")
	}
	if _, ok := arr[0]["role"]; !ok {
		t.Fatal("role should remain")
	}
}

func TestReplaySnipRemovalsAuto_embeddedUUID(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":[{"type":"text","text":"a"}],"rabbit_message_uuid":"m0"},
		{"role":"assistant","content":[{"type":"text","text":"b"}],"rabbit_message_uuid":"m1"},
		{"role":"user","content":[{"type":"text","text":"c"}],"rabbit_message_uuid":"m2"}
	]`)
	out, err := ReplaySnipRemovalsAuto(raw, []SnipRemovalEntry{
		{ID: "e", RemovedUUIDs: []string{"m1"}},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("len %d", len(arr))
	}
}

func TestReplaySnipRemovalsAuto_noEmbeddedErrors(t *testing.T) {
	raw := []byte(`[{"role":"user","content":[]}]`)
	_, err := ReplaySnipRemovalsAuto(raw, []SnipRemovalEntry{
		{ID: "e", RemovedUUIDs: []string{"nope"}},
	}, "")
	if !errors.Is(err, ErrSnipNoEmbeddedUUIDs) {
		t.Fatalf("got %v", err)
	}
}

func TestTranscriptMessageCount(t *testing.T) {
	msgs, err := buildThreeMessageTranscript(t)
	if err != nil {
		t.Fatal(err)
	}
	n, err := TranscriptMessageCount(msgs)
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestAnnotateTranscriptWithUUIDs(t *testing.T) {
	msgs, err := buildThreeMessageTranscript(t)
	if err != nil {
		t.Fatal(err)
	}
	out, err := AnnotateTranscriptWithUUIDs(msgs, []string{"a", "b", "c"}, "")
	if err != nil {
		t.Fatal(err)
	}
	m, err := BuildUUIDToIndexFromMessagesJSON(out, "")
	if err != nil || m["a"] != 0 || m["c"] != 2 {
		t.Fatalf("%+v err=%v", m, err)
	}
	_, err = AnnotateTranscriptWithUUIDs(msgs, []string{"x"}, "")
	if err == nil {
		t.Fatal("want len mismatch error")
	}
}

func TestStripMessageFieldsFromTranscriptJSON(t *testing.T) {
	raw := []byte(`[{"role":"user","rabbit_message_uuid":"u","extra":"x","content":[]}]`)
	out, err := StripMessageFieldsFromTranscriptJSON(raw, []string{RabbitMessageUUIDKey, "extra"})
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr[0]) != 2 {
		t.Fatalf("keys left: %d", len(arr[0]))
	}
}

func buildThreeMessageTranscript(t *testing.T) (json.RawMessage, error) {
	t.Helper()
	u1, err := InitialUserMessagesJSON("a")
	if err != nil {
		return nil, err
	}
	u1, err = AppendAssistantTextMessage(u1, "b")
	if err != nil {
		return nil, err
	}
	return AppendUserTextMessage(u1, "c")
}
