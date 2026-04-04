package compact

import (
	"encoding/json"
	"testing"
)

func TestGroupMessagesByApiRound_empty(t *testing.T) {
	if g := GroupMessagesByApiRound(nil); len(g) != 0 {
		t.Fatalf("%v", g)
	}
}

func TestGroupMessagesByApiRound_singleUser(t *testing.T) {
	ms := []ApiRoundMessage{{Type: "user"}}
	g := GroupMessagesByApiRound(ms)
	if len(g) != 1 || len(g[0]) != 1 {
		t.Fatalf("%v", g)
	}
}

func TestGroupMessagesByApiRound_splitsOnAssistantIDChange(t *testing.T) {
	ms := []ApiRoundMessage{
		{Type: "user"},
		assistantMsg("a1"),
		assistantMsg("a2"),
	}
	g := GroupMessagesByApiRound(ms)
	// TS: first assistant starts a new group after any non-empty current (user is own prefix group).
	if len(g) != 3 || len(g[0]) != 1 || len(g[1]) != 1 || len(g[2]) != 1 {
		t.Fatalf("got %d groups: %#v", len(g), g)
	}
	if g[0][0].Type != "user" || g[1][0].Message.ID != "a1" || g[2][0].Message.ID != "a2" {
		t.Fatalf("%#v", g)
	}
}

func TestGroupMessagesByApiRound_sameIDStaysOneGroup(t *testing.T) {
	ms := []ApiRoundMessage{
		{Type: "user"},
		assistantMsg("x"),
		assistantMsg("x"),
	}
	g := GroupMessagesByApiRound(ms)
	if len(g) != 2 || len(g[0]) != 1 || len(g[1]) != 2 {
		t.Fatalf("%v", g)
	}
}

func TestGroupMessagesByApiRound_JSONRoundTrip(t *testing.T) {
	raw := `[
	  {"type":"user","message":{}},
	  {"type":"assistant","message":{"id":"r1"}},
	  {"type":"assistant","message":{"id":"r1"}},
	  {"type":"assistant","message":{"id":"r2"}}
	]`
	var ms []ApiRoundMessage
	if err := json.Unmarshal([]byte(raw), &ms); err != nil {
		t.Fatal(err)
	}
	g := GroupMessagesByApiRound(ms)
	if len(g) != 3 {
		t.Fatalf("want 3 groups got %d %v", len(g), g)
	}
}

func assistantMsg(id string) ApiRoundMessage {
	var m ApiRoundMessage
	m.Type = "assistant"
	m.Message.ID = id
	return m
}
