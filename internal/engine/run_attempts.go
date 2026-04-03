package engine

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/compact"
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/querydeps"
)

const maxSubmitContinuationRounds = 8

func (e *Engine) loopDriver() query.LoopDriver {
	return query.LoopDriver{
		Deps: querydeps.Deps{
			Tools:     e.deps.Tools,
			Assistant: e.deps.Assistant,
			Turn:      e.deps.Turn,
		},
		Model:                e.model,
		MaxTokens:            e.maxTokens,
		Observe:              e.loopObservers(),
		HistorySnipMaxBytes:  features.HistorySnipMaxBytes(),
		HistorySnipMaxRounds: features.HistorySnipMaxRounds(),
		SnipCompactMaxBytes:  features.SnipCompactMaxBytes(),
		SnipCompactMaxRounds: features.SnipCompactMaxRounds(),
	}
}

// executeRunTurnLoopAttempts runs RunTurnLoop with optional RecoverStrategy second attempt (P5.1.3) and collapse-drain bookkeeping (H6).
func (e *Engine) executeRunTurnLoopAttempts(ctxLoop context.Context, st *query.LoopState, resolved string, maxAttempts int) (msgs json.RawMessage, succeeded bool, loopErr error) {
	d := e.loopDriver()
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt == 0 {
			if e.maxAssistantTurns > 0 {
				st.MaxTurns = e.maxAssistantTurns
			}
		} else {
			resetLoopStateForRetryAttempt(st)
		}

		var runErr error
		msgs, _, runErr = d.RunTurnLoop(ctxLoop, st, resolved)
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
				sum, exErr := e.compactExecutor(e.ctx, ph, msgs)
				e.trySend(EngineEvent{
					Kind:           EventKindCompactResult,
					CompactPhase:   ph.String(),
					CompactSummary: sum,
					Err:            exErr,
				})
				if exErr == nil {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
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
