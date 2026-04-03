package query

import (
	"encoding/json"
	"errors"
	"testing"
)

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
