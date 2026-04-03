package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMessage_JSONRoundTrip(t *testing.T) {
	m := Message{
		Role: RoleUser,
		Content: []ContentPiece{
			{Type: BlockTypeText, Text: "hello"},
			{Type: BlockTypeToolUse, ID: "toolu_1", Name: "bash", Input: json.RawMessage(`{"cmd":"ls"}`)},
		},
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var got Message
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Role != m.Role || len(got.Content) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Content[1].Name != "bash" {
		t.Fatalf("%+v", got.Content[1])
	}
	var wantInput, gotInput any
	if err := json.Unmarshal(m.Content[1].Input, &wantInput); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got.Content[1].Input, &gotInput); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wantInput, gotInput) {
		t.Fatalf("input: want %#v got %#v", wantInput, gotInput)
	}
}
