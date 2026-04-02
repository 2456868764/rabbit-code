package query

import "testing"

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
		{"start compact preserves counts", LoopState{TurnCount: 2, PendingTools: 1, InCompact: false, MaxTurns: 10, CompactCount: 0}, TranStartCompact, LoopState{TurnCount: 2, PendingTools: 1, InCompact: true, MaxTurns: 10, CompactCount: 1}},
		{"finish compact", LoopState{TurnCount: 2, PendingTools: 1, InCompact: true, MaxTurns: 10, CompactCount: 1}, TranFinishCompact, LoopState{TurnCount: 2, PendingTools: 1, InCompact: false, MaxTurns: 10, CompactCount: 1}},
		{"unknown is no-op", LoopState{TurnCount: 3, PendingTools: 1, InCompact: true, MaxTurns: 5, CompactCount: 2}, Transition("nope"), LoopState{TurnCount: 3, PendingTools: 1, InCompact: true, MaxTurns: 5, CompactCount: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ApplyTransition(tc.s, tc.tr)
			if got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}
