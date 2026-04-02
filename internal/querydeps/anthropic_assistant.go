package querydeps

import (
	"context"
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/anthropic"
)

// AnthropicAssistant implements StreamAssistant using internal/anthropic.Client (Phase 4 + Phase 5 bridge).
type AnthropicAssistant struct {
	Client           *anthropic.Client
	DefaultModel     string
	DefaultMaxTokens int
	Policy           anthropic.Policy
	// ExtraReadOptions are appended after context-derived options (tests / host hooks).
	ExtraReadOptions []anthropic.ReadAssistantOption
}

func (a *AnthropicAssistant) readOpts(ctx context.Context) []anthropic.ReadAssistantOption {
	var opts []anthropic.ReadAssistantOption
	if cb, ok := OnPromptCacheBreakFromContext(ctx); ok && cb != nil {
		opts = append(opts, anthropic.WithOnPromptCacheBreak(cb))
	}
	opts = append(opts, a.ExtraReadOptions...)
	return opts
}

// StreamAssistant calls PostMessagesStreamReadAssistant with messagesJSON as the Messages field.
func (a *AnthropicAssistant) StreamAssistant(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
	if a == nil || a.Client == nil {
		return "", ErrNilAnthropicClient
	}
	if model == "" {
		model = a.DefaultModel
	}
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	if maxTokens <= 0 {
		maxTokens = a.DefaultMaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	pol := a.Policy
	if pol.MaxAttempts == 0 {
		pol = anthropic.DefaultPolicy()
	}
	body := anthropic.MessagesStreamBody{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  json.RawMessage(messagesJSON),
	}
	text, _, err := a.Client.PostMessagesStreamReadAssistant(ctx, body, pol, a.readOpts(ctx)...)
	return text, err
}

// AssistantTurn implements TurnAssistant using streamed tool_use assembly (Phase 5).
func (a *AnthropicAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (TurnResult, error) {
	if a == nil || a.Client == nil {
		return TurnResult{}, ErrNilAnthropicClient
	}
	if model == "" {
		model = a.DefaultModel
	}
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	if maxTokens <= 0 {
		maxTokens = a.DefaultMaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	pol := a.Policy
	if pol.MaxAttempts == 0 {
		pol = anthropic.DefaultPolicy()
	}
	body := anthropic.MessagesStreamBody{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  json.RawMessage(messagesJSON),
	}
	turn, _, err := a.Client.PostMessagesStreamReadAssistantTurn(ctx, body, pol, a.readOpts(ctx)...)
	if err != nil {
		return TurnResult{}, err
	}
	out := TurnResult{
		Text:       turn.Text,
		StopReason: turn.StopReason,
	}
	for _, t := range turn.ToolUses {
		in := t.Input
		if len(in) == 0 {
			in = json.RawMessage(`{}`)
		}
		out.ToolUses = append(out.ToolUses, ToolUseCall{
			ID:    t.ID,
			Name:  t.Name,
			Input: in,
		})
	}
	return out, nil
}
