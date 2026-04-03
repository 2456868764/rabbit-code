package query

import (
	"encoding/json"
	"errors"
	"testing"
)

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
