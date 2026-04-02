package query

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/2456868764/rabbit-code/internal/querydeps"
)

// ErrMaxTurnsExceeded is returned when LoopState.MaxTurns > 0 and the cap is hit before another assistant call.
var ErrMaxTurnsExceeded = errors.New("query: max assistant turns exceeded")

// LoopDriver runs assistant and tool steps against querydeps.Deps (query.ts loop seed).
type LoopDriver struct {
	Deps      querydeps.Deps
	Model     string
	MaxTokens int
	Observe   *LoopObservers
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
	if st != nil {
		*st = ApplyTransition(*st, TranToolCallsDone)
	}
	return out, err
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
	msgs, err = InitialUserMessagesJSON(userText)
	if err != nil {
		return nil, "", err
	}
	model, max := d.modelAndMax()
	for {
		if st != nil && st.MaxTurns > 0 && st.TurnCount >= st.MaxTurns {
			return msgs, lastAssistantText, ErrMaxTurnsExceeded
		}
		turn, err := d.turner().AssistantTurn(ctx, model, max, msgs)
		if err != nil {
			return msgs, lastAssistantText, err
		}
		if len(turn.ToolUses) == 0 && turn.Text == "" {
			break
		}
		msgs, err = AppendAssistantTurnMessage(msgs, turn.Text, turn.ToolUses)
		if err != nil {
			return msgs, lastAssistantText, err
		}
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
				return msgs, lastAssistantText, err
			}
			if o := d.Observe; o != nil && o.OnToolDone != nil {
				o.OnToolDone(u.Name, u.ID, out)
			}
			results = append(results, ToolResultBlock{ToolUseID: u.ID, Content: string(out)})
		}
		msgs, err = AppendUserToolResultsMessage(msgs, results)
		if err != nil {
			return msgs, lastAssistantText, err
		}
	}
	return msgs, lastAssistantText, nil
}
