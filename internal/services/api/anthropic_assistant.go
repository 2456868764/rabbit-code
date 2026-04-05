package anthropic

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/compact"
	"github.com/2456868764/rabbit-code/internal/types"
)

// AnthropicAssistant implements StreamAssistant / TurnAssistant for the query loop (Phase 4 + Phase 5 bridge; TS: services/api + queryModelWithStreaming).
type AnthropicAssistant struct {
	Client           *Client
	DefaultModel     string
	DefaultMaxTokens int
	Policy           Policy
	// ExtraReadOptions are appended after context-derived options (tests / host hooks).
	ExtraReadOptions []ReadAssistantOption
	// MicrocompactBuffer optional; MarkToolsSentToAPIState after successful stream (microCompact.ts markToolsSentToAPIState).
	MicrocompactBuffer MicrocompactAPIStateMarker
	// SystemPrompt when non-empty is sent as the Messages API "system" string (memdir loadMemoryPrompt analogue, H8).
	SystemPrompt string
	// APIContextManagementOpts optional; when nil, GetAPIContextManagement uses zero options (thinking off unless set here).
	APIContextManagementOpts *compact.APIContextManagementOptions
	// ForkCompactSummary optional runForkedAgent / cache-sharing analogue: same inputs as compact.ts fork path; return raw assistant text or error to fall back to streaming.
	ForkCompactSummary func(ctx context.Context, summaryUserJSON []byte, transcriptJSON []byte) (assistantText string, err error)
	// ForkPartialCompactSummary optional cache-sharing for partial compact: messagesJSON is the Messages API array from BuildPartialCompactStreamRequestMessagesJSON (same body as the streaming path).
	ForkPartialCompactSummary func(ctx context.Context, messagesJSON []byte) (assistantText string, err error)
	// CompactToolsJSON optional Messages API tools for StreamCompactSummary; nil uses compact.DefaultCompactStreamingToolsJSON(features.CompactStreamingToolSearchEnabled()).
	CompactToolsJSON json.RawMessage
	// CompactStreamExtraBetas optional betas appended to compact stream body (e.g. tool-search beta when host enables deferred tools).
	CompactStreamExtraBetas []string
	// SessionActivityPing optional; with RemoteSendKeepalivesEnabled, called on CompactKeepAliveInterval during StreamCompactSummaryDetailed (compact.ts streamCompactSummary + sessionActivity).
	SessionActivityPing func(ctx context.Context)
	// CompactKeepAliveInterval defaults to 30s when <= 0.
	CompactKeepAliveInterval time.Duration
}

func (a *AnthropicAssistant) readOpts(ctx context.Context) []ReadAssistantOption {
	var opts []ReadAssistantOption
	if cb, ok := OnPromptCacheBreakFromContext(ctx); ok && cb != nil {
		opts = append(opts, WithOnPromptCacheBreak(cb))
	}
	opts = append(opts, a.ExtraReadOptions...)
	return opts
}

func (a *AnthropicAssistant) streamBody(model string, maxTokens int, messagesJSON []byte) MessagesStreamBody {
	body := MessagesStreamBody{
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
		body.AnthropicBeta = append(body.AnthropicBeta, BetaCachedMicrocompactBody)
	}
	a.attachAPIContextManagement(model, &body)
	return body
}

func applyPerTurnTaskBudgetFromContext(ctx context.Context, body *MessagesStreamBody) {
	if body == nil {
		return
	}
	total, ok := perTurnTaskBudgetTotal(ctx)
	if !ok {
		return
	}
	if body.OutputConfig == nil {
		body.OutputConfig = &OutputConfig{}
	}
	body.OutputConfig.TaskBudget = &TaskBudgetParam{Type: "tokens", Total: total}
}

func (a *AnthropicAssistant) attachAPIContextManagement(model string, body *MessagesStreamBody) {
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
	if !ShouldAttachContextManagementBeta(m, a.Client.Provider) {
		return
	}
	raw, err := json.Marshal(cm)
	if err != nil {
		return
	}
	body.ContextManagement = raw
	body.AnthropicBeta = AppendBetaUnique(body.AnthropicBeta, BetaContextManagement)
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
		pol = DefaultPolicy()
	}
	body := a.streamBody(model, maxTokens, messagesJSON)
	applyPerTurnTaskBudgetFromContext(ctx, &body)
	text, _, err := a.Client.PostMessagesStreamReadAssistant(ctx, body, pol, a.readOpts(ctx)...)
	if err == nil {
		a.markMicrocompactAfterSuccessfulAPI()
	}
	return text, err
}

// AssistantTurn implements streamed tool_use assembly (Phase 5).
func (a *AnthropicAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (types.TurnResult, error) {
	if a == nil || a.Client == nil {
		return types.TurnResult{}, ErrNilAnthropicClient
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
		pol = DefaultPolicy()
	}
	body := a.streamBody(model, maxTokens, messagesJSON)
	applyPerTurnTaskBudgetFromContext(ctx, &body)
	turn, _, err := a.Client.PostMessagesStreamReadAssistantTurn(ctx, body, pol, a.readOpts(ctx)...)
	if err != nil {
		return types.TurnResult{}, err
	}
	a.markMicrocompactAfterSuccessfulAPI()
	out := types.TurnResult{
		Text:       turn.Text,
		StopReason: turn.StopReason,
	}
	for _, t := range turn.ToolUses {
		in := t.Input
		if len(in) == 0 {
			in = json.RawMessage(`{}`)
		}
		out.ToolUses = append(out.ToolUses, types.ToolUseCall{
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
