package query

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
)

// ErrMaxTurnsExceeded is returned when LoopState.MaxTurns > 0 and the cap is hit before another assistant call.
var ErrMaxTurnsExceeded = errors.New("query: max assistant turns exceeded")

// LoopDriver runs assistant and tool steps against Deps (query.ts loop seed).
type LoopDriver struct {
	Deps      Deps
	Model     string
	MaxTokens int
	// AgentID optional subagent / analytics id (query.ts toolUseContext.agentId).
	AgentID string
	// NonInteractive mirrors toolUseContext.options.isNonInteractiveSession when true.
	NonInteractive bool
	// SessionID optional mirror for toolUseContext / session analytics (H6).
	SessionID string
	// Debug mirrors toolUseContext.options.debug (H6).
	Debug   bool
	Observe *LoopObservers
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
	// InitialLastAssistantAt seeds LoopState.LastAssistantAt for cross-Submit time-based microcompact (engine session).
	InitialLastAssistantAt time.Time
	// MicrocompactEditBuffer optional; reset when time-based microcompact clears tool results (microCompact.ts).
	MicrocompactEditBuffer *compact.MicrocompactEditBuffer
	// TaskBudgetTotal if > 0 sets output_config.task_budget on each Messages API assistant call (QueryEngine.ts taskBudget).
	TaskBudgetTotal int
	// SkipCacheWrite when true remaps prompt-cache breakpoints like query.ts skipCacheWrite (claude.ts addCacheBreakpoints).
	SkipCacheWrite bool
}

func (d *LoopDriver) streamer() StreamAssistant {
	if d.Deps.Assistant != nil {
		return d.Deps.Assistant
	}
	return NoopStreamAssistant{}
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

func (d *LoopDriver) turner() TurnAssistant {
	if d.Deps.Turn != nil {
		return d.Deps.Turn
	}
	return StreamAsTurnAssistant(d.Deps.Assistant)
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
		return nil, ErrNoToolRunner
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
	if d.TaskBudgetTotal > 0 {
		ctx = anthropic.WithPerTurnTaskBudget(ctx, d.TaskBudgetTotal)
	}
	if st != nil {
		st.SnipTokensFreedAccum = 0
	}
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
		if st.LastAssistantAt.IsZero() && !d.InitialLastAssistantAt.IsZero() {
			st.LastAssistantAt = d.InitialLastAssistantAt
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			st.ToolUseContext.AbortSignalAborted = true
		}
		st.SetMessagesJSON(msgs)
	}
	model, max := d.modelAndMax()
	var turn TurnResult
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
			if n > 0 {
				id := NewSnipRemovalID()
				if st != nil {
					st.SnipRemovalLog = append(st.SnipRemovalLog, SnipRemovalEntry{
						ID:                  id,
						Kind:                SnipRemovalKindHistorySnip,
						RemovedMessageCount: n,
						BytesBefore:         len(msgs),
						BytesAfter:          len(newMsgs),
					})
				}
				if d.Observe != nil && d.Observe.OnHistorySnip != nil {
					d.Observe.OnHistorySnip(len(msgs), len(newMsgs), n, id)
				}
			}
			if n > 0 {
				if nt := EstimateTranscriptJSONTokens(newMsgs); prevTok > nt {
					delta := prevTok - nt
					snipTokensFreed += delta
					if st != nil {
						st.SnipTokensFreedAccum += delta
					}
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
			if n > 0 {
				id := NewSnipRemovalID()
				if st != nil {
					st.SnipRemovalLog = append(st.SnipRemovalLog, SnipRemovalEntry{
						ID:                  id,
						Kind:                SnipRemovalKindSnipCompact,
						RemovedMessageCount: n,
						BytesBefore:         len(msgs),
						BytesAfter:          len(newMsgs),
					})
				}
				if d.Observe != nil && d.Observe.OnSnipCompact != nil {
					d.Observe.OnSnipCompact(len(msgs), len(newMsgs), n, id)
				}
			}
			if n > 0 {
				if nt := EstimateTranscriptJSONTokens(newMsgs); prevTok > nt {
					delta := prevTok - nt
					snipTokensFreed += delta
					if st != nil {
						st.SnipTokensFreedAccum += delta
					}
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
		if st != nil {
			qs := d.QuerySource
			if s := strings.TrimSpace(st.ToolUseContext.QuerySource); s != "" {
				qs = s
			}
			model, _ := d.modelAndMax()
			out, _, mcChanged, _, mcErr := compact.MicrocompactMessagesAPIJSON(msgs, qs, time.Now(), st.LastAssistantAt, model, d.MicrocompactEditBuffer)
			if mcErr != nil {
				st.SetMessagesJSON(msgs)
				return msgs, lastAssistantText, mcErr
			}
			if mcChanged {
				msgs = json.RawMessage(out)
				st.SetMessagesJSON(msgs)
			}
		}
		if d.SkipCacheWrite {
			next, rerr := RemapPromptCacheBreakpointsForSkipCacheWrite(msgs)
			if rerr != nil {
				if st != nil {
					st.SetMessagesJSON(msgs)
				}
				return msgs, lastAssistantText, rerr
			}
			msgs = next
			if st != nil {
				st.SetMessagesJSON(msgs)
			}
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
			st.LastAssistantAt = time.Now()
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
			return msgs, lastAssistantText, ErrNoToolRunner
		}
		toolBlocks := make([]any, 0, len(turn.ToolUses))
		var followUp [][]any
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
				in := u.Input
				if len(in) == 0 {
					in = []byte("{}")
				}
				o.OnToolDone(u.Name, u.ID, in, out)
			}
			content := any(string(out))
			if u.Name == filereadtool.FileReadToolName {
				c, sup := filereadtool.MapReadResultForMessagesAPI(out, filereadtool.MapReadResultOptions{
					MainLoopModel: d.Model,
				})
				if c != nil {
					content = c
				}
				followUp = append(followUp, sup...)
			} else if u.Name == filewritetool.FileWriteToolName {
				if s := filewritetool.MapWriteToolResultForMessagesAPI(out); s != "" {
					content = s
				}
			}
			toolBlocks = append(toolBlocks, map[string]any{
				"type":        "tool_result",
				"tool_use_id": u.ID,
				"content":     content,
			})
		}
		msgs, err = AppendUserMessageContentBlocks(msgs, toolBlocks)
		if err != nil {
			st.SetMessagesJSON(msgs)
			return msgs, lastAssistantText, err
		}
		for _, fb := range followUp {
			msgs, err = AppendUserMessageContentBlocks(msgs, fb)
			if err != nil {
				st.SetMessagesJSON(msgs)
				return msgs, lastAssistantText, err
			}
		}
		st.SetMessagesJSON(msgs)
		if o := d.Observe; o != nil && o.OnAfterToolResults != nil {
			if err := o.OnAfterToolResults(ctx, st, msgs); err != nil {
				st.SetMessagesJSON(msgs)
				return msgs, lastAssistantText, err
			}
		}
		if st != nil {
			ResetLoopStateFieldsForNextQueryIteration(st)
			RecordLoopContinue(st, LoopContinue{Reason: ContinueReasonNextTurn})
		}
	}
	st.SetMessagesJSON(msgs)
	return msgs, lastAssistantText, nil
}

// LoopObservers receives optional callbacks for engine / TUI wiring (Phase 5).
type LoopObservers struct {
	OnAssistantText func(text string)
	OnToolStart     func(name, toolUseID string, input []byte)
	OnToolDone      func(name, toolUseID string, inputJSON, result []byte)
	OnToolError     func(name, toolUseID string, err error)
	// OnAfterToolResults runs after tool results are appended to the transcript and state is mirrored, before next-turn reset (query.ts skillPrefetch / taskSummary timing).
	OnAfterToolResults         func(ctx context.Context, st *LoopState, transcriptJSON json.RawMessage) error
	OnHistorySnip              func(bytesBefore, bytesAfter, rounds int, snipID string)
	OnSnipCompact              func(bytesBefore, bytesAfter, rounds int, snipID string)
	OnPromptCacheBreakRecovery func(phase string)
}

// ErrBlockingLimit is returned before the first assistant API call when transcript usage is at or past the
// manual-compact buffer limit (query.ts calculateTokenWarningState / isAtBlockingLimit synthetic PTL).
var ErrBlockingLimit = errors.New("query: blocking limit exceeded")

// BlockingLimitPreCheckApplies mirrors query.ts gates before the blocking-limit check (lines 628–635).
func BlockingLimitPreCheckApplies(querySource string, skipDueToPostCompactContinuation bool) bool {
	if skipDueToPostCompactContinuation {
		return false
	}
	qs := strings.TrimSpace(querySource)
	if qs == QuerySourceSessionMemory || qs == QuerySourceCompact || qs == QuerySourceExtractMemories {
		return false
	}
	if features.ReactiveCompactEnabled() && features.IsAutoCompactEnabled() {
		return false
	}
	if features.ContextCollapseEnabled() && features.IsAutoCompactEnabled() {
		return false
	}
	return true
}

// CheckBlockingLimitPreAssistant runs the numeric blocking ladder (auto_compact.ts calculateTokenWarningState).
func CheckBlockingLimitPreAssistant(
	model string,
	maxOutputTokens int,
	contextWindowTokens int,
	transcriptJSON []byte,
	snipTokensFreed int,
	querySource string,
	skipDueToPostCompactContinuation bool,
) error {
	if !BlockingLimitPreCheckApplies(querySource, skipDueToPostCompactContinuation) {
		return nil
	}
	tokenUsage := EstimateTranscriptJSONTokens(transcriptJSON) - snipTokensFreed
	if tokenUsage < 0 {
		tokenUsage = 0
	}
	if n, err := EstimateMessageTokensFromTranscriptJSON(transcriptJSON); err == nil && n > 0 {
		tokenUsage = n - snipTokensFreed
		if tokenUsage < 0 {
			tokenUsage = 0
		}
	}
	r := BuildHeadlessContextReport(transcriptJSON, model, maxOutputTokens, contextWindowTokens, tokenUsage, querySource)
	if r.TokenWarning.IsAtBlockingLimit {
		return ErrBlockingLimit
	}
	return nil
}

// PromptCacheBreakRecovery optionally produces a new transcript after trim+resend still returns
// ErrPromptCacheBreakDetected (H1: compact coordination).
type PromptCacheBreakRecovery func(ctx context.Context, msgs json.RawMessage) (next json.RawMessage, ok bool, err error)

const maxPromptCacheBreakCompactRounds = 2

func (d *LoopDriver) assistantTurnWithPromptCacheBreakHandling(ctx context.Context, st *LoopState, model string, max int, msgs json.RawMessage) (TurnResult, json.RawMessage, error) {
	turn, err := d.turner().AssistantTurn(ctx, model, max, msgs)
	if err == nil {
		return turn, msgs, nil
	}
	if !errors.Is(err, anthropic.ErrPromptCacheBreakDetected) {
		return TurnResult{}, msgs, err
	}

	if features.PromptCacheBreakTrimResendEnabled() {
		next, stripped, serr := StripCacheControlFromMessagesJSON(msgs)
		if serr != nil {
			return TurnResult{}, msgs, serr
		}
		if stripped {
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
		for round := 0; round < maxPromptCacheBreakCompactRounds; round++ {
			if !errors.Is(err, anthropic.ErrPromptCacheBreakDetected) {
				break
			}
			next, ok, rerr := d.PromptCacheBreakRecovery(ctx, msgs)
			if rerr != nil {
				return TurnResult{}, msgs, rerr
			}
			if !ok || len(bytes.TrimSpace(next)) == 0 {
				break
			}
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
			if err == nil {
				return turn, msgs, nil
			}
		}
	}

	return TurnResult{}, msgs, err
}
