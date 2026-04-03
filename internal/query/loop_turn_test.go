package query

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

func TestLoopDriver_RunTurnLoop_toolThenText_AC5_3(t *testing.T) {
	tr := &countingToolRunner{}
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{
			Text: "invoke",
			ToolUses: []querydeps.ToolUseCall{
				{ID: "id1", Name: "bash", Input: json.RawMessage(`{"cmd":"ls"}`)},
			},
		},
		{Text: "done"},
	}}
	d := LoopDriver{
		Deps: querydeps.Deps{
			Tools: tr,
			Turn:  turns,
		},
		Model:          "m",
		MaxTokens:      64,
		AgentID:        "agent-test",
		NonInteractive: true,
		SessionID:      "sess-1",
		Debug:          true,
		QuerySource:    "compact_agent",
	}
	st := LoopState{MaxTurns: 10}
	_, last, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if last != "done" {
		t.Fatalf("last %q", last)
	}
	if n := tr.count(); n != 1 {
		t.Fatalf("tool runs %d", n)
	}
	if st.TurnCount != 2 {
		t.Fatalf("turns %+v", st)
	}
	if st.ToolUseContext.AgentID != "agent-test" || !st.ToolUseContext.NonInteractive || st.ToolUseContext.MainLoopModel != "m" {
		t.Fatalf("ToolUseContext %+v", st.ToolUseContext)
	}
	if st.ToolUseContext.SessionID != "sess-1" || !st.ToolUseContext.Debug {
		t.Fatalf("ToolUseContext %+v", st.ToolUseContext)
	}
	if st.ToolUseContext.QuerySource != "compact_agent" {
		t.Fatalf("QuerySource %q", st.ToolUseContext.QuerySource)
	}
}

func TestLoopDriver_RunTurnLoop_maxTurns_blocksSecondAssistantAfterTools(t *testing.T) {
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{
			Text: "need_tool",
			ToolUses: []querydeps.ToolUseCall{
				{ID: "t1", Name: "bash", Input: json.RawMessage(`{}`)},
			},
		},
		{Text: "never"},
	}}
	tr := &countingToolRunner{}
	d := LoopDriver{Deps: querydeps.Deps{Tools: tr, Turn: turns}, Model: "m", MaxTokens: 8}
	st := LoopState{MaxTurns: 1}
	_, _, err := d.RunTurnLoop(context.Background(), &st, "x")
	if !errors.Is(err, ErrMaxTurnsExceeded) {
		t.Fatalf("got %v", err)
	}
	if st.TurnCount != 1 {
		t.Fatalf("%+v", st)
	}
	if tr.count() != 1 {
		t.Fatalf("tools %d", tr.count())
	}
}

func TestLoopDriver_RunTurnLoopFromMessages_equivalentToRunTurnLoop(t *testing.T) {
	seed, err := InitialUserMessagesJSON("hi")
	if err != nil {
		t.Fatal(err)
	}
	mk := func() *querydeps.SequenceTurnAssistant {
		return &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{{Text: "reply"}}}
	}
	d1 := LoopDriver{Deps: querydeps.Deps{Turn: mk()}, Model: "m", MaxTokens: 8}
	d2 := LoopDriver{Deps: querydeps.Deps{Turn: mk()}, Model: "m", MaxTokens: 8}
	st1, st2 := LoopState{}, LoopState{}
	out1, _, err := d1.RunTurnLoop(context.Background(), &st1, "hi")
	if err != nil {
		t.Fatal(err)
	}
	out2, _, err := d2.RunTurnLoopFromMessages(context.Background(), &st2, seed)
	if err != nil {
		t.Fatal(err)
	}
	if string(out1) != string(out2) {
		t.Fatalf("transcripts differ:\n%s\nvs\n%s", out1, out2)
	}
	if st1.TurnCount != st2.TurnCount || st1.TurnCount != 1 {
		t.Fatalf("st1=%+v st2=%+v", st1, st2)
	}
}

func TestLoopDriver_RunTurnLoopFromMessages_emptySeed(t *testing.T) {
	d := LoopDriver{Deps: querydeps.Deps{Turn: &querydeps.SequenceTurnAssistant{}}, Model: "m", MaxTokens: 8}
	_, _, err := d.RunTurnLoopFromMessages(context.Background(), &LoopState{}, json.RawMessage(`   `))
	if err == nil {
		t.Fatal("want error")
	}
}

type countingToolRunner struct {
	n  int
	mu sync.Mutex
}

func (c *countingToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return []byte(`{"ok":true}`), nil
}

func (c *countingToolRunner) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
