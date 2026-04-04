package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
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
		AutoCompactTracking:           &compact.AutoCompactTracking{TurnID: "x", ConsecutiveFailures: &v},
		MaxOutputTokensRecoveryCount:  1,
		HasAttemptedReactiveCompact:   true,
		MaxOutputTokensOverrideActive: true,
		MaxOutputTokensOverride:       8000,
		PendingToolUseSummary:         true,
		StopHookActive:                true,
		TurnCount:                     9,
		PendingTools:                  3,
		SnipRemovalLog: []query.SnipRemovalEntry{
			{ID: "s1", Kind: query.SnipRemovalKindHistorySnip, RemovedMessageCount: 1},
		},
		LastAssistantAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
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
	if len(st.SnipRemovalLog) != 1 || st.SnipRemovalLog[0].ID != "s1" {
		t.Fatalf("snip log: %+v", st.SnipRemovalLog)
	}
	if !st.LastAssistantAt.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("LastAssistantAt: %v", st.LastAssistantAt)
	}
}
