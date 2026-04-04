package querydeps

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// AnthropicAssistant implements StreamAssistant using internal/services/api Client (Phase 4 + Phase 5 bridge).
type AnthropicAssistant struct {
	Client           *anthropic.Client
	DefaultModel     string
	DefaultMaxTokens int
	Policy           anthropic.Policy
	// ExtraReadOptions are appended after context-derived options (tests / host hooks).
	ExtraReadOptions []anthropic.ReadAssistantOption
	// MicrocompactBuffer optional; MarkToolsSentToAPIState after successful stream (microCompact.ts markToolsSentToAPIState).
	MicrocompactBuffer MicrocompactAPIStateMarker
	// SystemPrompt when non-empty is sent as the Messages API "system" string (memdir loadMemoryPrompt analogue, H8).
	SystemPrompt string
	// APIContextManagementOpts optional; when nil, GetAPIContextManagement uses zero options (thinking off unless set here).
	APIContextManagementOpts *compact.APIContextManagementOptions
}

func (a *AnthropicAssistant) readOpts(ctx context.Context) []anthropic.ReadAssistantOption {
	var opts []anthropic.ReadAssistantOption
	if cb, ok := OnPromptCacheBreakFromContext(ctx); ok && cb != nil {
		opts = append(opts, anthropic.WithOnPromptCacheBreak(cb))
	}
	opts = append(opts, a.ExtraReadOptions...)
	return opts
}

func (a *AnthropicAssistant) streamBody(model string, maxTokens int, messagesJSON []byte) anthropic.MessagesStreamBody {
	body := anthropic.MessagesStreamBody{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  json.RawMessage(messagesJSON),
	}
	if a != nil {
		if s := strings.TrimSpace(a.SystemPrompt); s != "" {
			b, _ := json.Marshal(s)
			body.System = b
		}
	}
	if features.CachedMicrocompactEnabled() {
		body.AnthropicBeta = append(body.AnthropicBeta, anthropic.BetaCachedMicrocompactBody)
	}
	a.attachAPIContextManagement(model, &body)
	return body
}

func (a *AnthropicAssistant) attachAPIContextManagement(model string, body *anthropic.MessagesStreamBody) {
	if a == nil || a.Client == nil || body == nil {
		return
	}
	m := strings.TrimSpace(model)
	opts := compact.APIContextManagementOptions{}
	if a.APIContextManagementOpts != nil {
		opts = *a.APIContextManagementOpts
	}
	cm := compact.GetAPIContextManagement(opts)
	if cm == nil {
		return
	}
	if !anthropic.ShouldAttachContextManagementBeta(m, a.Client.Provider) {
		return
	}
	raw, err := json.Marshal(cm)
	if err != nil {
		return
	}
	body.ContextManagement = raw
	body.AnthropicBeta = anthropic.AppendBetaUnique(body.AnthropicBeta, anthropic.BetaContextManagement)
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
	body := a.streamBody(model, maxTokens, messagesJSON)
	text, _, err := a.Client.PostMessagesStreamReadAssistant(ctx, body, pol, a.readOpts(ctx)...)
	if err == nil {
		a.markMicrocompactAfterSuccessfulAPI()
	}
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
	body := a.streamBody(model, maxTokens, messagesJSON)
	turn, _, err := a.Client.PostMessagesStreamReadAssistantTurn(ctx, body, pol, a.readOpts(ctx)...)
	if err != nil {
		return TurnResult{}, err
	}
	a.markMicrocompactAfterSuccessfulAPI()
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

func (a *AnthropicAssistant) markMicrocompactAfterSuccessfulAPI() {
	if a == nil || a.MicrocompactBuffer == nil || !features.CachedMicrocompactEnabled() {
		return
	}
	a.MicrocompactBuffer.MarkToolsSentToAPIState()
}
