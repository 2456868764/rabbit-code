package query

import "testing"

func TestApplyTransition_table(t *testing.T) {
	cases := []struct {
		name string
		s    LoopState
		tr   Transition
		want LoopState
	}{
		{"receive increments turn", LoopState{TurnCount: 0, PendingTools: 0}, TranReceiveAssistant, LoopState{1, 0, false, 0}},
		{"schedule tools", LoopState{1, 0, false, 0}, TranScheduleTools, LoopState{1, 1, false, 0}},
		{"tool done decrements", LoopState{1, 2, false, 0}, TranToolCallsDone, LoopState{1, 1, false, 0}},
		{"tool done clamps at zero", LoopState{0, 0, false, 0}, TranToolCallsDone, LoopState{0, 0, false, 0}},
		{"start compact preserves counts", LoopState{2, 1, false, 10}, TranStartCompact, LoopState{2, 1, true, 10}},
		{"finish compact", LoopState{2, 1, true, 10}, TranFinishCompact, LoopState{2, 1, false, 10}},
		{"unknown is no-op", LoopState{3, 1, true, 5}, Transition("nope"), LoopState{3, 1, true, 5}},
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
