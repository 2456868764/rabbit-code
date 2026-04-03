package query

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/querydeps"
)

func TestLoopContinue_recordAndClear(t *testing.T) {
	var st LoopState
	RecordLoopContinue(&st, LoopContinue{Reason: ContinueReasonReactiveCompactRetry})
	if st.LoopContinue.Reason != ContinueReasonReactiveCompactRetry {
		t.Fatal()
	}
	ClearLoopContinue(&st)
	if !st.LoopContinue.Empty() {
		t.Fatal()
	}
}

func TestLoopDriver_RunTurnLoop_setsNextTurnContinueAfterTools(t *testing.T) {
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{
			Text:     "t",
			ToolUses: []querydeps.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}},
		},
		{Text: "done"},
	}}
	d := LoopDriver{
		Deps: querydeps.Deps{
			Tools: querydeps.BashStubToolRunner{},
			Turn:  turns,
		},
		Model: "m", MaxTokens: 8,
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if st.LoopContinue.Reason != ContinueReasonNextTurn {
		t.Fatalf("want next_turn, got %+v", st.LoopContinue)
	}
	if len(st.MessagesJSON) == 0 || !strings.Contains(string(st.MessagesJSON), "hi") {
		t.Fatalf("MessagesJSON mirror: %s", st.MessagesJSON)
	}
	if st.ToolUseContext.MainLoopModel != "m" {
		t.Fatalf("ToolUseContext: %+v", st.ToolUseContext)
	}
}

func TestLoopDriver_RunTurnLoop_noTools_doesNotSetNextTurn(t *testing.T) {
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{{Text: "only"}}}
	d := LoopDriver{Deps: querydeps.Deps{Turn: turns}, Model: "m", MaxTokens: 8}
	st := LoopState{}
	_, _, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if !st.LoopContinue.Empty() {
		t.Fatalf("expected empty continue, got %+v", st.LoopContinue)
	}
}
