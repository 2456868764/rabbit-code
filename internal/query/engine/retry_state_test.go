package engine

import (
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
)

func TestResetLoopStateForRetryAttempt_preservesH6Fields(t *testing.T) {
	v := 2
	st := &query.LoopState{
		MaxTurns:                      7,
		CompactCount:                  1,
		MessagesJSON:                  json.RawMessage(`[1]`),
		ToolUseContext:                query.ToolUseContextMirror{AgentID: "x", MainLoopModel: "m"},
		RecoveryAttempts:              2,
		RecoveryPhase:                 query.RecoveryPendingCompact,
		LoopContinue:                  query.LoopContinue{Reason: query.ContinueReasonSubmitRecoverRetry},
		AutoCompactTracking:           &query.AutoCompactTracking{TurnID: "x", ConsecutiveFailures: &v},
		MaxOutputTokensRecoveryCount:  1,
		HasAttemptedReactiveCompact:   true,
		MaxOutputTokensOverrideActive: true,
		MaxOutputTokensOverride:       8000,
		PendingToolUseSummary:         true,
		StopHookActive:                true,
		TurnCount:                     9,
		PendingTools:                  3,
	}
	resetLoopStateForRetryAttempt(st)
	if st.TurnCount != 0 || st.PendingTools != 0 {
		t.Fatalf("per-attempt counters should reset, got %+v", st)
	}
	if st.LoopContinue.Reason != query.ContinueReasonSubmitRecoverRetry {
		t.Fatalf("%+v", st.LoopContinue)
	}
	if st.AutoCompactTracking == nil || st.AutoCompactTracking.TurnID != "x" || *st.AutoCompactTracking.ConsecutiveFailures != 2 {
		t.Fatal()
	}
	if st.RecoveryPhase != query.RecoveryRetriedOnce {
		t.Fatalf("got %v", st.RecoveryPhase)
	}
	if string(st.MessagesJSON) != `[1]` || st.ToolUseContext.AgentID != "x" {
		t.Fatalf("messages/tool ctx: %+v", st)
	}
}
