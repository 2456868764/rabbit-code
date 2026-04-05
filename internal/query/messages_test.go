package query

import (
	"encoding/json"
	"testing"
)

func TestAppendUserThenAssistant(t *testing.T) {
	msgs, err := InitialUserMessagesJSON("hi")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err = AppendAssistantTextMessage(msgs, "hello")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err = AppendUserTextMessage(msgs, "again")
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(msgs, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 3 {
		t.Fatalf("len %d", len(arr))
	}
}

func TestAppendAssistantTurnMessage_toolUseShape(t *testing.T) {
	base, err := InitialUserMessagesJSON("u")
	if err != nil {
		t.Fatal(err)
	}
	out, err := AppendAssistantTurnMessage(base, "t", []ToolUseCall{
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

func TestAppendUserMessageContentBlocks_toolResultArrayContent(t *testing.T) {
	base, err := InitialUserMessagesJSON("u")
	if err != nil {
		t.Fatal(err)
	}
	out, err := AppendUserMessageContentBlocks(base, []any{
		map[string]any{
			"type":        "tool_result",
			"tool_use_id": "tid",
			"content": []any{
				map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":        "base64",
						"media_type": "image/png",
						"data":        "eA==",
					},
				},
			},
		},
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
	content := msgs[1]["content"].([]any)
	tr := content[0].(map[string]any)
	if tr["type"] != "tool_result" {
		t.Fatal(tr)
	}
}

