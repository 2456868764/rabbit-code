package query

import (
	"encoding/json"
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
