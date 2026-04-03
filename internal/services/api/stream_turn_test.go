package anthropic

import (
	"context"
	"strings"
	"testing"
)

func TestReadAssistantStreamTurn_textOnly(t *testing.T) {
	raw := "" +
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}` + "\n\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}` + "\n\n" +
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":2}}}` + "\n\n" +
		`data: {"type":"message_stop"}` + "\n\n"
	turn, u, err := ReadAssistantStreamTurn(context.Background(), strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if turn.Text != "Hello" || turn.StopReason != "end_turn" || len(turn.ToolUses) != 0 {
		t.Fatalf("%+v", turn)
	}
	if u.InputTokens != 1 || u.OutputTokens != 2 {
		t.Fatalf("%+v", u)
	}
}

func TestReadAssistantStreamTurn_toolThenText(t *testing.T) {
	raw := "" +
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tu1","name":"bash","input":{}}}` + "\n\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"cmd\""}}` + "\n\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":":\"ls\"}"}}` + "\n\n" +
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}` + "\n\n" +
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Done"}}` + "\n\n" +
		`data: {"type":"message_stop"}` + "\n\n"
	turn, _, err := ReadAssistantStreamTurn(context.Background(), strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(turn.ToolUses) != 1 || turn.ToolUses[0].ID != "tu1" || turn.ToolUses[0].Name != "bash" {
		t.Fatalf("%+v", turn)
	}
	if string(turn.ToolUses[0].Input) != `{"cmd":"ls"}` {
		t.Fatalf("input %s", turn.ToolUses[0].Input)
	}
	if turn.Text != "Done" {
		t.Fatalf("text %q", turn.Text)
	}
}
