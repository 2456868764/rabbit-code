package query

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

// ErrMaxTurnsExceeded is returned when LoopState.MaxTurns > 0 and the cap is hit before another assistant call.
var ErrMaxTurnsExceeded = errors.New("query: max assistant turns exceeded")

// LoopDriver runs assistant and tool steps against querydeps.Deps (query.ts loop seed).
type LoopDriver struct {
	Deps      querydeps.Deps
	Model     string
	MaxTokens int
	// AgentID optional subagent / analytics id (query.ts toolUseContext.agentId).
	AgentID string
	// NonInteractive mirrors toolUseContext.options.isNonInteractiveSession when true.
	NonInteractive bool
	// SessionID optional mirror for toolUseContext / session analytics (H6).
	SessionID string
	// Debug mirrors toolUseContext.options.debug (H6).
	Debug bool
	Observe   *LoopObservers
	// HistorySnipMaxBytes / HistorySnipMaxRounds implement P5.F.10 when both > 0 (engine sets from features).
	HistorySnipMaxBytes  int
	HistorySnipMaxRounds int
	// SnipCompactMaxBytes / SnipCompactMaxRounds implement P5.2.2 when both > 0 (RABBIT_CODE_SNIP_COMPACT).
	SnipCompactMaxBytes  int
	SnipCompactMaxRounds int
	// PromptCacheBreakRecovery optional compact transcript after trim+resend still sees cache break (H1).
	PromptCacheBreakRecovery PromptCacheBreakRecovery
	// QuerySource optional fork id for proactive autocompact gates (autoCompact.ts).
	QuerySource string
	// ContextWindowTokens if > 0 overrides env/model default for blocking-limit pre-check (query.ts).
	ContextWindowTokens int
}

func (d *LoopDriver) streamer() querydeps.StreamAssistant {
	if d.Deps.Assistant != nil {
		return d.Deps.Assistant
	}
	return querydeps.NoopStreamAssistant{}
}

func (d *LoopDriver) modelAndMax() (model string, maxTokens int) {
	model, maxTokens = d.Model, d.MaxTokens
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	return model, maxTokens
}

func (d *LoopDriver) turner() querydeps.TurnAssistant {
	if d.Deps.Turn != nil {
		return d.Deps.Turn
	}
	return querydeps.StreamAsTurnAssistant(d.Deps.Assistant)
}

// RunAssistantStep calls StreamAssistant and appends the assistant text message to the transcript JSON.
func (d *LoopDriver) RunAssistantStep(ctx context.Context, messagesJSON json.RawMessage) (assistantText string, out json.RawMessage, err error) {
	model, max := d.modelAndMax()
	text, err := d.streamer().StreamAssistant(ctx, model, max, messagesJSON)
	if err != nil {
		return "", nil, err
	}
	out, err = AppendAssistantTextMessage(messagesJSON, text)
	return text, out, err
}

// RunToolStep invokes Tools.RunTool and applies schedule/done transitions when st is non-nil.
func (d *LoopDriver) RunToolStep(ctx context.Context, st *LoopState, name string, input []byte) ([]byte, error) {
	if d.Deps.Tools == nil {
		return nil, querydeps.ErrNoToolRunner
	}
	if st != nil {
		*st = ApplyTransition(*st, TranScheduleTools)
	}
	out, err := d.Deps.Tools.RunTool(ctx, name, input)
	if err != nil {
		if st != nil {
			*st = ApplyTransition(*st, TranToolCallsDone)
		}
		return nil, err
	}
	if st != nil {
		*st = ApplyTransition(*st, TranToolCallsDone)
	}
	return out, nil
}

// RunAssistantChain performs N assistant-only steps (mock multi-turn without tool_calls parsing).
func (d *LoopDriver) RunAssistantChain(ctx context.Context, userText string, steps int) (final json.RawMessage, texts []string, err error) {
	msgs, err := InitialUserMessagesJSON(userText)
	if err != nil {
		return nil, nil, err
	}
	for i := 0; i < steps; i++ {
		txt, next, err := d.RunAssistantStep(ctx, msgs)
		if err != nil {
			return msgs, texts, err
		}
		texts = append(texts, txt)
		msgs = next
	}
	return msgs, texts, nil
}

// RunTurnLoop runs assistant turns until the model returns no tool uses, ctx is done, or MaxTurns is exceeded.
// When st is non-nil, TranReceiveAssistant is applied after each assistant message; RunToolStep applies tool transitions.
func (d *LoopDriver) RunTurnLoop(ctx context.Context, st *LoopState, userText string) (msgs json.RawMessage, lastAssistantText string, err error) {
	return d.runTurnLoop(ctx, st, userText, nil)
}

// RunTurnLoopFromMessages continues from an existing messages JSON transcript (e.g. after context-collapse drain + RecoverStrategy retry). seedMsgs must be non-empty.
func (d *LoopDriver) RunTurnLoopFromMessages(ctx context.Context, st *LoopState, seedMsgs json.RawMessage) (msgs json.RawMessage, lastAssistantText string, err error) {
	if len(bytes.TrimSpace(seedMsgs)) == 0 {
		return nil, "", errors.New("query: RunTurnLoopFromMessages: empty seed")
	}
	return d.runTurnLoop(ctx, st, "", seedMsgs)
}

func (d *LoopDriver) runTurnLoop(ctx context.Context, st *LoopState, userText string, seedMsgs json.RawMessage) (msgs json.RawMessage, lastAssistantText string, err error) {
	if len(seedMsgs) > 0 {
		buf := make([]byte, len(seedMsgs))
		copy(buf, seedMsgs)
		msgs = json.RawMessage(buf)
	} else {
		msgs, err = InitialUserMessagesJSON(userText)
		if err != nil {
			return nil, "", err
		}
	}
	if st != nil {
		st.ToolUseContext.AbortSignalAborted = false
		model0, _ := d.modelAndMax()
		st.ToolUseContext.MainLoopModel = model0
		st.ToolUseContext.AgentID = d.AgentID
		st.ToolUseContext.NonInteractive = d.NonInteractive
		st.ToolUseContext.SessionID = d.SessionID
		st.ToolUseContext.Debug = d.Debug
		st.ToolUseContext.QuerySource = d.QuerySource
		if errors.Is(ctx.Err(), context.Canceled) {
			st.ToolUseContext.AbortSignalAborted = true
		}
		st.SetMessagesJSON(msgs)
	}
	model, max := d.modelAndMax()
	var turn querydeps.TurnResult
	skipBlockingDueToContinuation := len(seedMsgs) > 0
	firstAssistant := true
	var snipTokensFreed int
	for {
		if st != nil && st.MaxTurns > 0 && st.TurnCount >= st.MaxTurns {
			st.SetMessagesJSON(msgs)
			return msgs, lastAssistantText, ErrMaxTurnsExceeded
		}
		if d.HistorySnipMaxBytes > 0 && d.HistorySnipMaxRounds > 0 {
			prevTok := EstimateTranscriptJSONTokens(msgs)
			newMsgs, n, err := TrimTranscriptPrefixWhileOverBudget(msgs, d.HistorySnipMaxBytes, d.HistorySnipMaxRounds)
			if err != nil {
				st.SetMessagesJSON(msgs)
				return msgs, lastAssistantText, err
			}
			if n > 0 && d.Observe != nil && d.Observe.OnHistorySnip != nil {
				d.Observe.OnHistorySnip(len(msgs), len(newMsgs), n)
			}
			if n > 0 {
				if nt := EstimateTranscriptJSONTokens(newMsgs); prevTok > nt {
					snipTokensFreed += prevTok - nt
				}
			}
			msgs = newMsgs
			st.SetMessagesJSON(msgs)
		}
		if d.SnipCompactMaxBytes > 0 && d.SnipCompactMaxRounds > 0 {
			prevTok := EstimateTranscriptJSONTokens(msgs)
			newMsgs, n, err := TrimTranscriptPrefixWhileOverBudget(msgs, d.SnipCompactMaxBytes, d.SnipCompactMaxRounds)
			if err != nil {
				st.SetMessagesJSON(msgs)
				return msgs, lastAssistantText, err
			}
			if n > 0 && d.Observe != nil && d.Observe.OnSnipCompact != nil {
				d.Observe.OnSnipCompact(len(msgs), len(newMsgs), n)
			}
			if n > 0 {
				if nt := EstimateTranscriptJSONTokens(newMsgs); prevTok > nt {
					snipTokensFreed += prevTok - nt
				}
			}
			msgs = newMsgs
			st.SetMessagesJSON(msgs)
		}
		if firstAssistant {
			qs := d.QuerySource
			if st != nil {
				if s := strings.TrimSpace(st.ToolUseContext.QuerySource); s != "" {
					qs = s
				}
			}
			cw := d.ContextWindowTokens
			if err := CheckBlockingLimitPreAssistant(model, max, cw, msgs, snipTokensFreed, qs, skipBlockingDueToContinuation); err != nil {
				if st != nil {
					st.SetMessagesJSON(msgs)
				}
				return msgs, lastAssistantText, err
			}
			firstAssistant = false
		}
		turn, msgs, err = d.assistantTurnWithPromptCacheBreakHandling(ctx, st, model, max, msgs)
		if err != nil {
			if st != nil && errors.Is(err, context.Canceled) {
				st.ToolUseContext.AbortSignalAborted = true
			}
			st.SetMessagesJSON(msgs)
			return msgs, lastAssistantText, err
		}
		if len(turn.ToolUses) == 0 && turn.Text == "" {
			break
		}
		msgs, err = AppendAssistantTurnMessage(msgs, turn.Text, turn.ToolUses)
		if err != nil {
			st.SetMessagesJSON(msgs)
			return msgs, lastAssistantText, err
		}
		st.SetMessagesJSON(msgs)
		if st != nil {
			*st = ApplyTransition(*st, TranReceiveAssistant)
			st.LastStopReason = turn.StopReason
		}
		lastAssistantText = turn.Text
		if o := d.Observe; o != nil && o.OnAssistantText != nil && turn.Text != "" {
			o.OnAssistantText(turn.Text)
		}
		if len(turn.ToolUses) == 0 {
			break
		}
		if d.Deps.Tools == nil {
			st.SetMessagesJSON(msgs)
			return msgs, lastAssistantText, querydeps.ErrNoToolRunner
		}
		results := make([]ToolResultBlock, 0, len(turn.ToolUses))
		for _, u := range turn.ToolUses {
			if o := d.Observe; o != nil && o.OnToolStart != nil {
				in := u.Input
				if len(in) == 0 {
					in = []byte("{}")
				}
				o.OnToolStart(u.Name, u.ID, in)
			}
			out, err := d.RunToolStep(ctx, st, u.Name, u.Input)
			if err != nil {
				if o := d.Observe; o != nil && o.OnToolError != nil {
					o.OnToolError(u.Name, u.ID, err)
				}
				st.SetMessagesJSON(msgs)
				return msgs, lastAssistantText, err
			}
			if o := d.Observe; o != nil && o.OnToolDone != nil {
				o.OnToolDone(u.Name, u.ID, out)
			}
			results = append(results, ToolResultBlock{ToolUseID: u.ID, Content: string(out)})
		}
		msgs, err = AppendUserToolResultsMessage(msgs, results)
		if err != nil {
			st.SetMessagesJSON(msgs)
			return msgs, lastAssistantText, err
		}
		st.SetMessagesJSON(msgs)
		if st != nil {
			ResetLoopStateFieldsForNextQueryIteration(st)
			RecordLoopContinue(st, LoopContinue{Reason: ContinueReasonNextTurn})
		}
	}
	st.SetMessagesJSON(msgs)
	return msgs, lastAssistantText, nil
}
