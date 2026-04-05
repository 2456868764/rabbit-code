package anthropic

import (
	"context"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
)

func TestReadWebSearchAssistantBlocks_textServerToolResult(t *testing.T) {
	raw := "" +
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"Start"}}` + "\n\n" +
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" end"}}` + "\n\n" +
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"server_tool_use","id":"st1","name":"web_search"}}` + "\n\n" +
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"q\"}"}}` + "\n\n" +
		`data: {"type":"content_block_start","index":2,"content_block":{"type":"web_search_tool_result","tool_use_id":"st1","content":[{"title":"T","url":"https://u"}]}}` + "\n\n" +
		`data: {"type":"message_delta","delta":{"usage":{"input_tokens":3,"output_tokens":4}}}` + "\n\n" +
		`data: {"type":"message_stop"}` + "\n\n"
	blocks, u, err := ReadWebSearchAssistantBlocks(context.Background(), strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if u.InputTokens != 3 || u.OutputTokens != 4 {
		t.Fatalf("usage %+v", u)
	}
	if len(blocks) != 3 {
		t.Fatalf("len(blocks)=%d", len(blocks))
	}
	got, err := websearchtool.MakeOutputFromContentBlocks(blocks, "q", 0.1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got)=%d %#v", len(got), got)
	}
	if got[0] != "Start end" {
		t.Fatalf("got0 %q", got[0])
	}
	blk, ok := got[1].(websearchtool.SearchResultBlock)
	if !ok || blk.ToolUseID != "st1" || len(blk.Content) != 1 || blk.Content[0].URL != "https://u" {
		t.Fatalf("got1 %+v ok=%v", got[1], ok)
	}
}
