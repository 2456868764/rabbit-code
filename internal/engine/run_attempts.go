package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/services/compact"
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/querydeps"
)

const maxSubmitContinuationRounds = 8

func (e *Engine) loopDriver() query.LoopDriver {
	d := query.LoopDriver{
		Deps: querydeps.Deps{
			Tools:     e.deps.Tools,
			Assistant: e.deps.Assistant,
			Turn:      e.deps.Turn,
		},
		Model:            e.model,
		MaxTokens:        e.maxTokens,
		AgentID:          e.agentID,
		NonInteractive:   e.nonInteractive,
		SessionID:        e.sessionID,
		Debug:            e.debug,
		Observe:          e.loopObservers(),
		HistorySnipMaxBytes:  features.HistorySnipMaxBytes(),
		HistorySnipMaxRounds: features.HistorySnipMaxRounds(),
		SnipCompactMaxBytes:  features.SnipCompactMaxBytes(),
		SnipCompactMaxRounds: features.SnipCompactMaxRounds(),
	}
	if features.PromptCacheBreakAutoCompactEnabled() && e.compactExecutor != nil {
		d.PromptCacheBreakRecovery = e.promptCacheBreakCompactRecovery
	}
	return d
}

func (e *Engine) promptCacheBreakCompactRecovery(ctx context.Context, msgs json.RawMessage) (json.RawMessage, bool, error) {
	if e.compactExecutor == nil {
		return nil, false, nil
	}
	ph := compact.RunIdle.Next(false, true)
	_, next, err := e.compactExecutor(ctx, ph, msgs)
	if err != nil {
		return nil, false, err
	}
	next = bytes.TrimSpace(next)
	if len(next) == 0 {
		return nil, false, nil
	}
	return json.RawMessage(append([]byte(nil), next...)), true, nil
}

// executeRunTurnLoopAttempts runs RunTurnLoop with optional RecoverStrategy second attempt (P5.1.3) and collapse-drain bookkeeping (H6).
func (e *Engine) executeRunTurnLoopAttempts(ctxLoop context.Context, st *query.LoopState, resolved string, maxAttempts int) (msgs json.RawMessage, succeeded bool, loopErr error) {
	d := e.loopDriver()
	var retrySeedMsgs json.RawMessage
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt == 0 {
			if e.maxAssistantTurns > 0 {
				st.MaxTurns = e.maxAssistantTurns
			}
		} else {
			resetLoopStateForRetryAttempt(st)
		}

		drainedForRetry := false
		var compactRetrySeed []byte
		var runErr error
		if len(retrySeedMsgs) > 0 {
			msgs, _, runErr = d.RunTurnLoopFromMessages(ctxLoop, st, retrySeedMsgs)
			retrySeedMsgs = nil
		} else {
			msgs, _, runErr = d.RunTurnLoop(ctxLoop, st, resolved)
		}
		if runErr == nil {
			return msgs, true, nil
		}
		loopErr = runErr
		if errors.Is(runErr, context.Canceled) || errors.Is(e.ctx.Err(), context.Canceled) {
			return msgs, false, loopErr
		}
		st.HadStreamError = true
		kind, rec := classifyAnthropicError(runErr)
		st.LastAPIErrorKind = kind
		if rec {
			st.RecoveryAttempts++
			if st.RecoveryPhase == query.RecoveryNone {
				st.RecoveryPhase = query.RecoveryPendingCompact
			}
		}
		if rec && kind == string(anthropic.KindPromptTooLong) &&
			features.ContextCollapseEnabled() &&
			e.contextCollapseDrain != nil &&
			st.LoopContinue.Reason != query.ContinueReasonCollapseDrainRetry {
			trimmed, committed, ok := e.contextCollapseDrain(e.ctx, st, msgs)
			if ok && committed > 0 {
				msgs = trimmed
				drainedForRetry = true
				query.RecordLoopContinue(st, query.LoopContinue{
					Reason:    query.ContinueReasonCollapseDrainRetry,
					Committed: committed,
				})
			}
		}
		if rec && e.suggestCompactOnRecoverableError {
			ph := compact.RunIdle.Next(true, false)
			e.trySend(EngineEvent{
				Kind:               EventKindCompactSuggest,
				CompactPhase:       ph.String(),
				SuggestAutoCompact: true,
			})
			if e.compactExecutor != nil {
				execPh := compact.ExecutorPhaseAfterSchedule(ph)
				sum, next, exErr := e.compactExecutor(e.ctx, execPh, msgs)
				resPh := compact.ResultPhaseAfterCompactExecutor(execPh, exErr)
				e.trySend(EngineEvent{
					Kind:           EventKindCompactResult,
					CompactPhase:   resPh.String(),
					CompactSummary: sum,
					Err:            exErr,
				})
				if exErr == nil {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
					if len(bytes.TrimSpace(next)) > 0 {
						compactRetrySeed = append([]byte(nil), next...)
						msgs = json.RawMessage(append([]byte(nil), next...))
						st.SetMessagesJSON(msgs)
					}
				}
			}
		}
		willRetry := attempt+1 < maxAttempts && e.recoverStrategy != nil && e.recoverStrategy(e.ctx, *st, runErr)
		if willRetry {
			if kind == string(anthropic.KindMaxOutputTokens) {
				st.MaxOutputTokensRecoveryCount++
				query.RecordLoopContinue(st, query.LoopContinue{
					Reason:  query.ContinueReasonMaxOutputTokensRecovery,
					Attempt: st.MaxOutputTokensRecoveryCount,
				})
			} else {
				query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonSubmitRecoverRetry})
			}
			if len(compactRetrySeed) > 0 {
				retrySeedMsgs = json.RawMessage(append([]byte(nil), compactRetrySeed...))
			} else if drainedForRetry {
				retrySeedMsgs = json.RawMessage(append([]byte(nil), msgs...))
			}
			continue
		}
		e.trySend(EngineEvent{
			Kind:               EventKindError,
			Err:                runErr,
			APIErrorKind:       kind,
			RecoverableCompact: rec,
		})
		return msgs, false, loopErr
	}
	return msgs, false, loopErr
}
