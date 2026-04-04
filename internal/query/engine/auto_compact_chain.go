package engine

import (
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// runCompactSuggestAfterSuccessfulTurn mirrors autoCompact.ts autoCompactIfNeeded scheduling tail on the main thread:
// advisor + proactive token gate + optional session-memory compact + compact suggest/executor + loop continue flags.
// Proactive auto + session memory run through compact.AutoCompactIfNeeded (single entry). Reactive compact stays separate.
// Prompt-cache-break reactive compact runs earlier in the same Submit (engine.go); not duplicated here.
func (e *Engine) runCompactSuggestAfterSuccessfulTurn(st *query.LoopState, msgs json.RawMessage) json.RawMessage {
	if features.DisableCompact() {
		return msgs
	}
	advisorAuto, react := false, false
	if e.compactAdvisor != nil {
		advisorAuto, react = e.compactAdvisor(*st, msgs)
	}
	cw := e.contextWindowTokens
	if cw <= 0 {
		cw = features.ContextWindowTokensForModel(e.model)
	}
	cw = features.ApplyAutoCompactWindowCap(cw)
	autocompactTok := query.EstimateTranscriptJSONTokens(msgs)
	if n, err := query.EstimateMessageTokensFromTranscriptJSON(msgs); err == nil && n > 0 {
		autocompactTok = n
	}
	if compact.AfterTurnReactiveCompactSuggested(msgs, features.ReactiveCompactMinTranscriptBytes(), features.ReactiveCompactMinEstimatedTokens(), st != nil && st.HasAttemptedReactiveCompact) {
		react = true
	}

	snipFreed := 0
	if st != nil {
		snipFreed = st.SnipTokensFreedAccum
	}
	var afterSM func(string, string)
	if e.afterSessionMemoryCompactSuccess != nil {
		afterSM = func(qs, aid string) {
			e.afterSessionMemoryCompactSuccess(e.ctx, qs, aid)
		}
	}
	acRes, _ := compact.AutoCompactIfNeeded(compact.AutoCompactIfNeededInput{
		Ctx:                         e.ctx,
		TranscriptJSON:              msgs,
		Model:                       e.model,
		MaxOutputTokens:             e.maxTokens,
		ContextWindowTokens:         cw,
		QuerySource:                 st.ToolUseContext.QuerySource,
		AgentID:                     st.ToolUseContext.AgentID,
		SnipTokensFreed:             snipFreed,
		TokenUsage:                  autocompactTok,
		TrackingConsecutiveFailures: e.autoCompactConsecutiveFailures,
		ForceAuto:                   advisorAuto,
		SessionMemoryCompact:        e.sessionMemoryCompact,
		AfterSessionMemorySuccess:   afterSM,
	})

	proactiveAuto := !acRes.Skipped && !acRes.SessionMemoryApplied && acRes.RunLegacyAutoCompact

	if acRes.SessionMemoryApplied {
		msgs = acRes.NewTranscript
		st.SetMessagesJSON(msgs)
		ph := compact.RunIdle.Next(true, false)
		execPh := compact.ExecutorPhaseAfterSchedule(ph)
		e.trySend(EngineEvent{
			Kind:               EventKindCompactSuggest,
			CompactPhase:       ph.String(),
			SuggestAutoCompact: true,
		})
		resPh := compact.ResultPhaseAfterCompactExecutor(execPh, nil)
		e.noteAutoCompactExecutorOutcome(st, true, nil)
		e.trySend(EngineEvent{
			Kind:           EventKindCompactResult,
			CompactPhase:   resPh.String(),
			CompactSummary: "session_memory_compact",
			Err:            nil,
		})
		query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonAutoCompactExecuted})
		react = compact.AfterTurnReactiveCompactSuggested(msgs, features.ReactiveCompactMinTranscriptBytes(), features.ReactiveCompactMinEstimatedTokens(), st != nil && st.HasAttemptedReactiveCompact)
		e.afterCompactSuccess(st)
	}

	if proactiveAuto || react {
		phase := compact.RunIdle.Next(proactiveAuto, react)
		e.trySend(EngineEvent{
			Kind:                   EventKindCompactSuggest,
			CompactPhase:           phase.String(),
			SuggestAutoCompact:     proactiveAuto,
			SuggestReactiveCompact: react,
		})
		if e.compactExecutor != nil {
			execPh := compact.ExecutorPhaseAfterSchedule(phase)
			ctxCompact := compact.ContextWithExecutorSuggestMeta(e.ctx, compact.ExecutorSuggestMeta{
				AutoCompact:     proactiveAuto,
				ReactiveCompact: react,
			})
			sum, _, exErr := e.compactExecutor(ctxCompact, execPh, msgs)
			resPh := compact.ResultPhaseAfterCompactExecutor(execPh, exErr)
			e.noteAutoCompactExecutorOutcome(st, proactiveAuto, exErr)
			e.trySend(EngineEvent{
				Kind:           EventKindCompactResult,
				CompactPhase:   resPh.String(),
				CompactSummary: sum,
				Err:            exErr,
			})
			if exErr == nil {
				if react {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
					st.HasAttemptedReactiveCompact = true
				} else if proactiveAuto {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonAutoCompactExecuted})
				}
				e.afterCompactSuccess(st)
			}
		}
	}
	return msgs
}
