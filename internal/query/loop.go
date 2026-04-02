package query

import (
	"context"
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/querydeps"
)

// LoopDriver runs assistant and tool steps against querydeps.Deps (query.ts loop seed).
type LoopDriver struct {
	Deps      querydeps.Deps
	Model     string
	MaxTokens int
}

func (d *LoopDriver) streamer() querydeps.StreamAssistant {
	if d.Deps.Assistant != nil {
		return d.Deps.Assistant
	}
	return querydeps.NoopStreamAssistant{}
}

// RunAssistantStep calls StreamAssistant and appends the assistant text message to the transcript JSON.
func (d *LoopDriver) RunAssistantStep(ctx context.Context, messagesJSON json.RawMessage) (assistantText string, out json.RawMessage, err error) {
	model, max := d.Model, d.MaxTokens
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	if max <= 0 {
		max = 1024
	}
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
