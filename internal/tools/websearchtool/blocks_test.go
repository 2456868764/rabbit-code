package websearchtool

import (
	"encoding/json"
	"testing"
)

func TestMakeOutputFromContentBlocks(t *testing.T) {
	blocks := []json.RawMessage{
		json.RawMessage(`{"type":"text","text":"Intro "}`),
		json.RawMessage(`{"type":"server_tool_use","id":"st1"}`),
		json.RawMessage(`{"type":"web_search_tool_result","tool_use_id":"tu1","content":[{"title":"A","url":"https://a"}]}`),
		json.RawMessage(`{"type":"text","text":" outro"}`),
	}
	got, err := MakeOutputFromContentBlocks(blocks, "q", 1.2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len %d %#v", len(got), got)
	}
	if got[0] != "Intro" {
		t.Fatalf("got0 %q", got[0])
	}
	blk, ok := got[1].(SearchResultBlock)
	if !ok || blk.ToolUseID != "tu1" || len(blk.Content) != 1 || blk.Content[0].URL != "https://a" {
		t.Fatalf("block %+v", got[1])
	}
	if got[2] != "outro" {
		t.Fatalf("got2 %q", got[2])
	}
}

func TestMakeOutputFromContentBlocks_errorContent(t *testing.T) {
	blocks := []json.RawMessage{
		json.RawMessage(`{"type":"web_search_tool_result","tool_use_id":"x","content":{"error_code":"busy"}}`),
	}
	got, err := MakeOutputFromContentBlocks(blocks, "q", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatal(got)
	}
	s, ok := got[0].(string)
	if !ok || s != "Web search error: busy" {
		t.Fatalf("got %q ok=%v", s, ok)
	}
}

func TestMakeOutputFromContentBlocks_errorContent_missingCode(t *testing.T) {
	blocks := []json.RawMessage{
		json.RawMessage(`{"type":"web_search_tool_result","tool_use_id":"x","content":{}}`),
	}
	got, err := MakeOutputFromContentBlocks(blocks, "q", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatal(got)
	}
	s, ok := got[0].(string)
	if !ok || s != "Web search error: undefined" {
		t.Fatalf("got %q ok=%v", s, ok)
	}
}

func TestWebSearchToolSchemaFromInput(t *testing.T) {
	s := WebSearchToolSchemaFromInput(Input{
		Query:          "q",
		AllowedDomains: []string{"a.com"},
	})
	if s.Type != "web_search_20250305" || s.Name != "web_search" || s.MaxUses != MaxSearchUses {
		t.Fatalf("%+v", s)
	}
	if len(s.AllowedDomains) != 1 || s.BlockedDomains != nil {
		t.Fatalf("%+v", s)
	}
}
