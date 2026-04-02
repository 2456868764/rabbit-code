package query

import (
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/querydeps"
)

func TestAppendAssistantTurnMessage_toolUseShape(t *testing.T) {
	base, err := InitialUserMessagesJSON("u")
	if err != nil {
		t.Fatal(err)
	}
	out, err := AppendAssistantTurnMessage(base, "t", []querydeps.ToolUseCall{
		{ID: "x", Name: "bash", Input: json.RawMessage(`{"k":1}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var msgs []map[string]any
	if err := json.Unmarshal(out, &msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len %d", len(msgs))
	}
}
