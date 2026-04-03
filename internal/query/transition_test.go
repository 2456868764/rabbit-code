package query

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestApplyTransition_table(t *testing.T) {
	cases := []struct {
		name string
		s    LoopState
		tr   Transition
		want LoopState
	}{
		{"receive increments turn", LoopState{TurnCount: 0, PendingTools: 0}, TranReceiveAssistant, LoopState{TurnCount: 1, PendingTools: 0, InCompact: false, MaxTurns: 0, CompactCount: 0}},
		{"schedule tools", LoopState{TurnCount: 1, PendingTools: 0}, TranScheduleTools, LoopState{TurnCount: 1, PendingTools: 1, InCompact: false, MaxTurns: 0, CompactCount: 0}},
		{"tool done decrements", LoopState{TurnCount: 1, PendingTools: 2}, TranToolCallsDone, LoopState{TurnCount: 1, PendingTools: 1, InCompact: false, MaxTurns: 0, CompactCount: 0}},
		{"tool done clamps at zero", LoopState{TurnCount: 0, PendingTools: 0}, TranToolCallsDone, LoopState{TurnCount: 0, PendingTools: 0, InCompact: false, MaxTurns: 0, CompactCount: 0}},
		{"finish compact", LoopState{TurnCount: 2, PendingTools: 1, InCompact: true, MaxTurns: 10, CompactCount: 1}, TranFinishCompact, LoopState{TurnCount: 2, PendingTools: 1, InCompact: false, MaxTurns: 10, CompactCount: 1}},
		{"unknown is no-op", LoopState{TurnCount: 3, PendingTools: 1, InCompact: true, MaxTurns: 5, CompactCount: 2}, Transition("nope"), LoopState{TurnCount: 3, PendingTools: 1, InCompact: true, MaxTurns: 5, CompactCount: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ApplyTransition(tc.s, tc.tr)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestApplyTransition_preservesMessagesJSONAndToolUseContext(t *testing.T) {
	raw := json.RawMessage(`[{"role":"user"}]`)
	s := LoopState{
		TurnCount:    0,
		MessagesJSON: raw,
		ToolUseContext: ToolUseContextMirror{
			AgentID: "ag", MainLoopModel: "m", NonInteractive: true, SessionID: "s", Debug: true, QuerySource: "q",
		},
	}
	got := ApplyTransition(s, TranReceiveAssistant)
	if string(got.MessagesJSON) != string(raw) {
		t.Fatalf("MessagesJSON: %s", got.MessagesJSON)
	}
	if got.ToolUseContext.AgentID != "ag" || got.ToolUseContext.MainLoopModel != "m" || !got.ToolUseContext.NonInteractive ||
		got.ToolUseContext.SessionID != "s" || !got.ToolUseContext.Debug || got.ToolUseContext.QuerySource != "q" {
		t.Fatalf("%+v", got.ToolUseContext)
	}
}

func TestApplyTransition_TranStartCompact_setsAutoCompactTracking(t *testing.T) {
	s := LoopState{TurnCount: 2, PendingTools: 1, InCompact: false, MaxTurns: 10, CompactCount: 0}
	got := ApplyTransition(s, TranStartCompact)
	if !got.InCompact || got.CompactCount != 1 {
		t.Fatalf("compact flags: %+v", got)
	}
	if got.AutoCompactTracking == nil || !got.AutoCompactTracking.Compacted ||
		got.AutoCompactTracking.TurnCounter != 2 || got.AutoCompactTracking.TurnID != "autocompact:1" {
		t.Fatalf("AutoCompactTracking: %+v", got.AutoCompactTracking)
	}
}
