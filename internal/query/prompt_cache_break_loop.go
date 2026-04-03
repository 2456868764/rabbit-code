package query

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/querydeps"
)

// PromptCacheBreakRecovery optionally produces a new transcript after trim+resend still returns
// ErrPromptCacheBreakDetected (H1: compact coordination). If ok is false, the loop returns the last error.
type PromptCacheBreakRecovery func(ctx context.Context, msgs json.RawMessage) (next json.RawMessage, ok bool, err error)

func (d *LoopDriver) assistantTurnWithPromptCacheBreakHandling(ctx context.Context, st *LoopState, model string, max int, msgs json.RawMessage) (querydeps.TurnResult, json.RawMessage, error) {
	turn, err := d.turner().AssistantTurn(ctx, model, max, msgs)
	if err == nil {
		return turn, msgs, nil
	}
	if !errors.Is(err, anthropic.ErrPromptCacheBreakDetected) {
		return querydeps.TurnResult{}, msgs, err
	}

	if features.PromptCacheBreakTrimResendEnabled() {
		next, stripped, serr := StripCacheControlFromMessagesJSON(msgs)
		if serr == nil && stripped {
			if st != nil {
				RecordLoopContinue(st, LoopContinue{Reason: ContinueReasonPromptCacheBreakTrimResend})
			}
			if o := d.Observe; o != nil && o.OnPromptCacheBreakRecovery != nil {
				o.OnPromptCacheBreakRecovery("trim_resend")
			}
			msgs = next
			if st != nil {
				st.SetMessagesJSON(msgs)
			}
			turn, err = d.turner().AssistantTurn(ctx, model, max, msgs)
			if err == nil {
				return turn, msgs, nil
			}
		}
	}

	if errors.Is(err, anthropic.ErrPromptCacheBreakDetected) && d.PromptCacheBreakRecovery != nil && features.PromptCacheBreakAutoCompactEnabled() {
		next, ok, rerr := d.PromptCacheBreakRecovery(ctx, msgs)
		if rerr != nil {
			return querydeps.TurnResult{}, msgs, rerr
		}
		if ok && len(bytes.TrimSpace(next)) > 0 {
			if st != nil {
				RecordLoopContinue(st, LoopContinue{Reason: ContinueReasonPromptCacheBreakCompactRetry})
			}
			if o := d.Observe; o != nil && o.OnPromptCacheBreakRecovery != nil {
				o.OnPromptCacheBreakRecovery("compact_retry")
			}
			msgs = json.RawMessage(append([]byte(nil), next...))
			if st != nil {
				st.SetMessagesJSON(msgs)
			}
			turn, err = d.turner().AssistantTurn(ctx, model, max, msgs)
			return turn, msgs, err
		}
	}

	return querydeps.TurnResult{}, msgs, err
}
