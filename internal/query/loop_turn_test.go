package query

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/2456868764/rabbit-code/internal/querydeps"
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
		Model:     "m",
		MaxTokens: 64,
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
