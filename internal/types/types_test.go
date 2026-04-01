package types

import (
	"encoding/json"
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
	if got.Content[1].Name != "bash" || string(got.Content[1].Input) != `{"cmd":"ls"}` {
		t.Fatalf("%+v", got.Content[1])
	}
}
