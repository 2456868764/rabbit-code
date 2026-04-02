package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Client is a minimal Messages API HTTP client (services/api/claude.ts + client.ts).
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	Provider   Provider
	BetaHeader string
	// bedrockBodyBetas are betas that must go in JSON anthropic_beta (not HTTP header) on Bedrock.
	bedrockBodyBetas []string
	// SessionID is sent as X-Claude-Code-Session-Id (services/api/client.ts defaultHeaders).
	SessionID string
	// ExtraHeaders are merged on each request (x-app, session id, anti-distillation, etc.).
	ExtraHeaders http.Header
	// OnStreamUsage is invoked once after PostMessagesStreamReadAssistant finishes reading the stream
	// (P4.4.1 / cost hook); called with the final UsageDelta (may be zero).
	OnStreamUsage func(UsageDelta)
	// ThinkingAccumulator receives thinking_delta fragments when non-nil (interleaved thinking streams).
	ThinkingAccumulator *strings.Builder
	// CompactionAccumulator receives compaction_delta fragments when non-nil (context-management streams).
	CompactionAccumulator *strings.Builder
	// ToolInputJSONByBlock maps content_block index → accumulator for input_json_delta partial_json (parallel tool calls use distinct indices).
	ToolInputJSONByBlock map[int]*strings.Builder
}

// NewClient returns a client with sane defaults.
func NewClient(rt http.RoundTripper) *Client {
	return &Client{
		HTTPClient: &http.Client{Transport: rt},
		BaseURL:    BaseURL(DetectProvider()),
		Provider:   DetectProvider(),
		ExtraHeaders: http.Header{
			"x-app": []string{"cli"},
		},
	}
}

// SetBetaNames sets BetaHeader and, for ProviderBedrock, stores body-only betas (SplitBetasForBedrock / extraBodyParams).
func (c *Client) SetBetaNames(names []string) {
	if c.Provider == ProviderBedrock {
		h, e := SplitBetasForBedrock(names)
		c.BetaHeader = MergeBetaHeader(h)
		c.bedrockBodyBetas = append([]string(nil), e...)
		return
	}
	c.bedrockBodyBetas = nil
	c.BetaHeader = MergeBetaHeader(names)
}

func (c *Client) messagesURL(body MessagesStreamBody) string {
	base := strings.TrimRight(c.BaseURL, "/")
	switch c.Provider {
	case ProviderBedrock:
		return base + BedrockStreamPath(body.Model)
	case ProviderVertex:
		if envVertexProjectID() != "" {
			return base + VertexStreamPath(envVertexProjectID(), vertexRegion(), body.Model)
		}
		return base + MessagesPath(c.Provider)
	default:
		return base + MessagesPath(c.Provider)
	}
}

// vertexStreamJSONBody matches @anthropic-ai/vertex-sdk: model moves to path, body gets anthropic_version.
type vertexStreamJSONBody struct {
	MaxTokens        int             `json:"max_tokens"`
	Stream           bool            `json:"stream"`
	Messages         json.RawMessage `json:"messages"`
	OutputConfig     *OutputConfig   `json:"output_config,omitempty"`
	AnthropicBeta    []string        `json:"anthropic_beta,omitempty"`
	AnthropicVersion string          `json:"anthropic_version"`
}

func (c *Client) marshalMessagesStreamJSON(body MessagesStreamBody) ([]byte, error) {
	if c.Provider == ProviderBedrock && len(c.bedrockBodyBetas) > 0 && len(body.AnthropicBeta) == 0 {
		body.AnthropicBeta = append([]string(nil), c.bedrockBodyBetas...)
	}
	body.Stream = true
	if c.Provider == ProviderVertex && envVertexProjectID() != "" {
		vb := vertexStreamJSONBody{
			MaxTokens:        body.MaxTokens,
			Stream:           true,
			Messages:         body.Messages,
			OutputConfig:     body.OutputConfig,
			AnthropicBeta:    append([]string(nil), body.AnthropicBeta...),
			AnthropicVersion: VertexDefaultAnthropicVersion,
		}
		return json.Marshal(vb)
	}
	return json.Marshal(body)
}

// MessagesStreamBody is the JSON body for POST .../messages with stream:true (subset).
type MessagesStreamBody struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
	Messages  json.RawMessage `json:"messages"`
	// OutputConfig carries task_budget etc. (claude.ts configureTaskBudgetParams).
	OutputConfig *OutputConfig `json:"output_config,omitempty"`
	// AnthropicBeta is sent as JSON "anthropic_beta" (Bedrock: betas in BEDROCK_EXTRA_PARAMS_HEADERS; 1P often uses header only).
	AnthropicBeta []string `json:"anthropic_beta,omitempty"`
}

// PostMessagesStream starts a streaming request. Caller must close resp.Body.
func (c *Client) PostMessagesStream(ctx context.Context, body MessagesStreamBody, pol Policy) (*http.Response, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	raw, err := c.marshalMessagesStreamJSON(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.messagesURL(body), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	payload := raw
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	betaVal := c.BetaHeader
	if body.OutputConfig != nil && body.OutputConfig.TaskBudget != nil {
		betaVal = MergeBetaHeaderAppend(betaVal, BetaTaskBudgets)
	}
	for _, extra := range features.OAuthBetaAppendNames() {
		betaVal = MergeBetaHeaderAppend(betaVal, extra)
	}
	if betaVal != "" {
		req.Header.Set("anthropic-beta", betaVal)
	}
	if sid := strings.TrimSpace(c.SessionID); sid != "" {
		req.Header.Set("X-Claude-Code-Session-Id", sid)
	}
	for k, vv := range MergedCustomHeadersFromEnv() {
		if len(vv) > 0 {
			req.Header.Set(k, vv[0])
		}
	}
	if features.AdditionalProtectionHeader() {
		req.Header.Set("x-anthropic-additional-protection", "true")
	}
	if name, val, ok := features.AntiDistillationRequestHeader(); ok {
		req.Header.Set(name, val)
	}
	if name, val, ok := features.NativeAttestationRequestHeader(); ok {
		req.Header.Set(name, val)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", UserAgent())
	}
	for k, v := range c.ExtraHeaders {
		req.Header[k] = append([]string(nil), v...)
	}
	tr := c.HTTPClient.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}
	return DoRequest(ctx, tr, req, pol)
}

// PostMessagesStreamReadAssistant posts, reads the SSE body to completion, closes the response, then
// calls OnStreamUsage when set (P4.1.3 / P4.4.1).
func (c *Client) PostMessagesStreamReadAssistant(ctx context.Context, body MessagesStreamBody, pol Policy) (string, UsageDelta, error) {
	resp, err := c.PostMessagesStream(ctx, body, pol)
	if err != nil {
		return "", UsageDelta{}, err
	}
	defer resp.Body.Close()
	var ropts []ReadAssistantOption
	if c.ThinkingAccumulator != nil {
		ropts = append(ropts, WithThinkingAccumulator(c.ThinkingAccumulator))
	}
	if c.CompactionAccumulator != nil {
		ropts = append(ropts, WithCompactionAccumulator(c.CompactionAccumulator))
	}
	if len(c.ToolInputJSONByBlock) > 0 {
		ropts = append(ropts, WithToolInputAccumulators(c.ToolInputJSONByBlock))
	}
	text, u, err := ReadAssistantStream(ctx, resp.Body, ropts...)
	if err != nil {
		return text, u, err
	}
	if c.OnStreamUsage != nil {
		c.OnStreamUsage(u)
	}
	return text, u, nil
}

type readAssistantConfig struct {
	thinking           *strings.Builder
	compaction         *strings.Builder
	toolInputByBlock   map[int]*strings.Builder
	onPromptCacheBreak func()
}

// ReadAssistantOption configures ReadAssistantStream.
type ReadAssistantOption func(*readAssistantConfig)

// WithThinkingAccumulator appends thinking_delta text from content_block_delta events into acc.
func WithThinkingAccumulator(acc *strings.Builder) ReadAssistantOption {
	return func(c *readAssistantConfig) {
		c.thinking = acc
	}
}

// WithCompactionAccumulator appends compaction_delta text from content_block_delta events into acc.
func WithCompactionAccumulator(acc *strings.Builder) ReadAssistantOption {
	return func(c *readAssistantConfig) {
		c.compaction = acc
	}
}

// WithToolInputAccumulators appends input_json_delta partial_json into the builder for each event index present in byIndex.
func WithToolInputAccumulators(byIndex map[int]*strings.Builder) ReadAssistantOption {
	return func(c *readAssistantConfig) {
		c.toolInputByBlock = byIndex
	}
}

// WithOnPromptCacheBreak runs fn when PROMPT_CACHE_BREAK_DETECTION matches an SSE error before returning ErrPromptCacheBreakDetected (AC4-F3 hook).
func WithOnPromptCacheBreak(fn func()) ReadAssistantOption {
	return func(c *readAssistantConfig) {
		c.onPromptCacheBreak = fn
	}
}

// ReadAssistantStream consumes SSE until message_stop; returns full text and last usage.
func ReadAssistantStream(ctx context.Context, body io.Reader, opts ...ReadAssistantOption) (text string, usage UsageDelta, err error) {
	var cfg readAssistantConfig
	for _, o := range opts {
		o(&cfg)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan StreamEvent, StreamBufferCapacity)
	errCh := make(chan error, 1)
	go func() {
		errCh <- StreamEvents(ctx, body, ch)
	}()

	var b strings.Builder
	var u UsageDelta
	var haveUsage bool
	for ev := range ch {
		switch ParseEventType(ev.JSON) {
		case "content_block_delta":
			_ = AppendTextDelta(ev.JSON, &b)
			if cfg.thinking != nil {
				_ = AppendThinkingDelta(ev.JSON, cfg.thinking)
			}
			if cfg.compaction != nil {
				_ = AppendCompactionDelta(ev.JSON, cfg.compaction)
			}
			if len(cfg.toolInputByBlock) > 0 {
				_ = AppendInputJSONDelta(ev.JSON, cfg.toolInputByBlock)
			}
		case "message_delta":
			if ud, ok := ParseUsageDelta(ev.JSON); ok {
				u = ud
				haveUsage = true
			}
		case "error":
			var evErr error
			if IsPromptCacheBreakStreamJSON(ev.JSON) {
				if cfg.onPromptCacheBreak != nil {
					cfg.onPromptCacheBreak()
				}
				evErr = ErrPromptCacheBreakDetected
			} else {
				evErr = fmt.Errorf("stream error event: %s", string(ev.JSON))
			}
			cancel()
			for range ch {
			}
			<-errCh
			return b.String(), u, evErr
		}
	}

	streamErr := <-errCh
	if streamErr != nil && streamErr != io.EOF {
		return b.String(), u, streamErr
	}
	if !haveUsage {
		u = UsageDelta{}
	}
	return b.String(), u, nil
}
