package engine

import (
	"bytes"
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// runCompactSuggestAfterSuccessfulTurn mirrors autoCompact.ts autoCompactIfNeeded scheduling tail on the main thread:
// advisor + proactive token gate + optional session-memory compact + compact suggest/executor + loop continue flags.
// Prompt-cache-break reactive compact runs earlier in the same Submit (engine.go); not duplicated here.
func (e *Engine) runCompactSuggestAfterSuccessfulTurn(st *query.LoopState, msgs json.RawMessage) json.RawMessage {
	if features.DisableCompact() {
		return msgs
	}
	auto, react := false, false
	if e.compactAdvisor != nil {
		auto, react = e.compactAdvisor(*st, msgs)
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
	if compact.AfterTurnProactiveAutocompactFromUsage(autocompactTok, e.model, e.maxTokens, cw, st.ToolUseContext.QuerySource, e.autoCompactCircuitTripped()) {
		auto = true
	}
	if compact.AfterTurnReactiveCompactSuggested(msgs, features.ReactiveCompactMinTranscriptBytes(), features.ReactiveCompactMinEstimatedTokens(), st != nil && st.HasAttemptedReactiveCompact) {
		react = true
	}

	if auto && e.sessionMemoryCompact != nil {
		th := compact.AutoCompactThresholdForProactive(e.model, e.maxTokens, cw)
		if th > 0 {
			next, ok, smErr := e.sessionMemoryCompact(e.ctx, st.ToolUseContext.AgentID, e.model, th, msgs)
			switch {
			case smErr != nil:
				// Legacy auto path may still run.
			case ok && len(bytes.TrimSpace(next)) > 0:
				msgs = json.RawMessage(append([]byte(nil), next...))
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
				auto = false
				react = compact.AfterTurnReactiveCompactSuggested(msgs, features.ReactiveCompactMinTranscriptBytes(), features.ReactiveCompactMinEstimatedTokens(), st != nil && st.HasAttemptedReactiveCompact)
				e.afterCompactSuccess(st)
			}
		}
	}

	if auto || react {
		phase := compact.RunIdle.Next(auto, react)
		e.trySend(EngineEvent{
			Kind:                   EventKindCompactSuggest,
			CompactPhase:           phase.String(),
			SuggestAutoCompact:     auto,
			SuggestReactiveCompact: react,
		})
		if e.compactExecutor != nil {
			execPh := compact.ExecutorPhaseAfterSchedule(phase)
			ctxCompact := compact.ContextWithExecutorSuggestMeta(e.ctx, compact.ExecutorSuggestMeta{
				AutoCompact:     auto,
				ReactiveCompact: react,
			})
			sum, _, exErr := e.compactExecutor(ctxCompact, execPh, msgs)
			resPh := compact.ResultPhaseAfterCompactExecutor(execPh, exErr)
			e.noteAutoCompactExecutorOutcome(st, auto, exErr)
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
				} else if auto {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonAutoCompactExecuted})
				}
				e.afterCompactSuccess(st)
			}
		}
	}
	return msgs
}
