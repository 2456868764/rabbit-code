package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

// LLMSpanInfo is emitted before DoRequest when LLMSpanStart is set (tracing / newContext parity).
type LLMSpanInfo struct {
	Model        string
	Streaming    bool
	BodyBytes    int
	EffortToken  string
	RequestID    string
	QuerySource  QuerySource
}

// LLMSpanEndInfo is emitted after DoRequest returns when LLMSpanEnd is set.
type LLMSpanEndInfo struct {
	Streaming  bool
	StatusCode int
	Err        error
	Duration   time.Duration
}

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
	// LLMDeriveContext optionally wraps ctx per Messages request (OpenTelemetry / newContext analogue).
	LLMDeriveContext func(ctx context.Context, model string, streaming bool) context.Context
	// LLMSpanStart / LLMSpanEnd bracket DoRequest for LLM spans.
	LLMSpanStart func(ctx context.Context, info LLMSpanInfo)
	LLMSpanEnd   func(ctx context.Context, info LLMSpanEndInfo)

	sessionLatchMu           sync.Mutex
	sessionLatchedBetas      []string
	thinkingClearLatched     bool
	afkHeaderLatched         bool
	cacheEditingHeaderLatched bool
	envSessionLatchesLoaded  bool

	transportMu  sync.Mutex
	cachedBaseRT http.RoundTripper
	cachedWrapRT http.RoundTripper
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
	MaxTokens         int             `json:"max_tokens"`
	Stream            bool            `json:"stream"`
	Messages          json.RawMessage `json:"messages"`
	System            json.RawMessage `json:"system,omitempty"`
	Tools             json.RawMessage `json:"tools,omitempty"`
	ToolChoice        json.RawMessage `json:"tool_choice,omitempty"`
	Thinking          json.RawMessage `json:"thinking,omitempty"`
	Temperature       *float64        `json:"temperature,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	OutputConfig      *OutputConfig   `json:"output_config,omitempty"`
	AnthropicBeta     []string        `json:"anthropic_beta,omitempty"`
	ContextManagement json.RawMessage `json:"context_management,omitempty"`
	AnthropicVersion  string          `json:"anthropic_version"`
	AntiDistillation  []string        `json:"anti_distillation,omitempty"`
	Speed             string          `json:"speed,omitempty"`
}

// EnvRabbitMessagesAPISpeed sets MessagesStreamBody.Speed when non-empty (claude.ts fast-mode body.speed).
const EnvRabbitMessagesAPISpeed = "RABBIT_CODE_MESSAGES_API_SPEED"

func (c *Client) mergeStreamingBody(body MessagesStreamBody, pol Policy) MessagesStreamBody {
	if c.Provider == ProviderBedrock && len(c.bedrockBodyBetas) > 0 && len(body.AnthropicBeta) == 0 {
		body.AnthropicBeta = append([]string(nil), c.bedrockBodyBetas...)
	}
	if features.AntiDistillationFakeToolsInBody() {
		body.AntiDistillation = []string{"fake_tools"}
	}
	if len(bytes.TrimSpace(body.Metadata)) == 0 {
		if meta, err := BuildMessagesAPIMetadata(c); err == nil && len(meta) > 0 {
			body.Metadata = meta
		}
	}
	if strings.TrimSpace(body.Speed) == "" {
		if s := strings.TrimSpace(os.Getenv(EnvRabbitMessagesAPISpeed)); s != "" {
			body.Speed = s
		}
	}
	body.Speed = c.effectiveBodySpeed(body, pol)
	return body
}

func (c *Client) effectiveBodySpeed(body MessagesStreamBody, pol Policy) string {
	if c == nil {
		return strings.TrimSpace(body.Speed)
	}
	speed := strings.TrimSpace(body.Speed)
	wantFast := pol.FastMode || strings.EqualFold(speed, "fast")
	if !wantFast {
		return speed
	}
	if c.Provider != ProviderAnthropic {
		return ""
	}
	if !features.FastModeOrganizationAvailable() || IsFastModeCooldown() {
		return ""
	}
	return "fast"
}

func (c *Client) marshalMessagesStreamJSON(body MessagesStreamBody, pol Policy) ([]byte, error) {
	return c.marshalMessagesJSON(body, true, pol)
}

func (c *Client) marshalMessagesJSON(body MessagesStreamBody, stream bool, pol Policy) ([]byte, error) {
	body = c.mergeStreamingBody(body, pol)
	body.Stream = stream
	extra := extraBodyParamsFromEnv()
	if c.Provider == ProviderVertex && envVertexProjectID() != "" {
		vb := vertexStreamJSONBody{
			MaxTokens:         body.MaxTokens,
			Stream:            stream,
			Messages:          body.Messages,
			System:            body.System,
			Tools:             append(json.RawMessage(nil), body.Tools...),
			ToolChoice:        append(json.RawMessage(nil), body.ToolChoice...),
			Thinking:          append(json.RawMessage(nil), body.Thinking...),
			Temperature:       body.Temperature,
			Metadata:          append(json.RawMessage(nil), body.Metadata...),
			OutputConfig:      body.OutputConfig,
			AnthropicBeta:     append([]string(nil), body.AnthropicBeta...),
			ContextManagement: append(json.RawMessage(nil), body.ContextManagement...),
			AnthropicVersion:  VertexDefaultAnthropicVersion,
			AntiDistillation:  append([]string(nil), body.AntiDistillation...),
			Speed:             body.Speed,
		}
		raw, err := json.Marshal(vb)
		if err != nil {
			return nil, err
		}
		return applyExtraBodyMerge(raw, extra)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return applyExtraBodyMerge(raw, extra)
}

func applyExtraBodyMerge(raw []byte, extra map[string]any) ([]byte, error) {
	if len(extra) == 0 {
		return raw, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	mergeExtraBodyIntoMap(m, extra)
	return json.Marshal(m)
}

func (c *Client) effectiveTransport() http.RoundTripper {
	tr := c.HTTPClient.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}
	if !features.DisableKeepAliveOnECONNRESETEnabled() {
		return tr
	}
	// Avoid toggling DisableKeepAlives on the process-wide default transport.
	if tr == http.DefaultTransport {
		return tr
	}
	c.transportMu.Lock()
	defer c.transportMu.Unlock()
	if c.cachedWrapRT != nil && c.cachedBaseRT == tr {
		return c.cachedWrapRT
	}
	c.cachedBaseRT = tr
	if t, ok := tr.(*http.Transport); ok {
		c.cachedWrapRT = newKeepAliveResetTransport(t)
	} else {
		c.cachedWrapRT = tr
	}
	return c.cachedWrapRT
}

// mergeBodyAnthropicBetasIntoHeader appends JSON body anthropic_beta entries to the HTTP anthropic-beta header
// (claude.ts parity). On Bedrock, betas in BedrockExtraParamsBetas stay body-only and are not duplicated in the header.
func mergeBodyAnthropicBetasIntoHeader(header string, bodyBetas []string, p Provider) string {
	out := header
	for _, b := range bodyBetas {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		if p == ProviderBedrock {
			if _, bodyOnly := BedrockExtraParamsBetas[b]; bodyOnly {
				continue
			}
		}
		out = MergeBetaHeaderAppend(out, b)
	}
	return out
}

// MessagesStreamBody is the JSON body for POST .../messages with stream:true (subset).
type MessagesStreamBody struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
	Messages  json.RawMessage `json:"messages"`
	// System is optional plain-text system prompt (loadMemoryPrompt / memdir.ts analogue).
	System json.RawMessage `json:"system,omitempty"`
	// OutputConfig carries task_budget etc. (claude.ts configureTaskBudgetParams).
	OutputConfig *OutputConfig `json:"output_config,omitempty"`
	// AnthropicBeta is sent as JSON "anthropic_beta" (Bedrock: betas in BEDROCK_EXTRA_PARAMS_HEADERS; 1P often uses header only).
	AnthropicBeta []string `json:"anthropic_beta,omitempty"`
	// AntiDistillation is merged when RABBIT_CODE_ANTI_DISTILLATION_CC + RABBIT_CODE_ANTI_DISTILLATION_FAKE_TOOLS (claude.ts getExtraBodyParams).
	AntiDistillation []string `json:"anti_distillation,omitempty"`
	// ContextManagement is sent when getAPIContextManagement returns edits and the context-management beta is active (apiMicrocompact.ts / claude.ts).
	ContextManagement json.RawMessage `json:"context_management,omitempty"`
	// Tools is optional tool definitions (compact.ts streamCompactSummary Read / ToolSearch path).
	Tools json.RawMessage `json:"tools,omitempty"`
	// ToolChoice optional (WebSearchTool.call Haiku path: {type:"tool",name:"web_search"}).
	ToolChoice json.RawMessage `json:"tool_choice,omitempty"`
	// Thinking optional; WebSearch inner call uses {"type":"disabled"} when forcing small model + tool_choice.
	Thinking json.RawMessage `json:"thinking,omitempty"`
	// Temperature when thinking disabled (claude.ts parity for inner web search request).
	Temperature *float64 `json:"temperature,omitempty"`
	// Metadata is optional; when empty, marshal merges BuildMessagesAPIMetadata(c) (claude.ts getAPIMetadata).
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// Speed optional fast-mode body field (claude.ts speed; e.g. RABBIT_CODE_MESSAGES_API_SPEED=fast).
	Speed string `json:"speed,omitempty"`
}

func (c *Client) messagesBetaHeader(body MessagesStreamBody, pol Policy) string {
	betaVal := c.BetaHeader
	if body.OutputConfig != nil && body.OutputConfig.TaskBudget != nil {
		betaVal = MergeBetaHeaderAppend(betaVal, BetaTaskBudgets)
	}
	for _, extra := range features.OAuthBetaAppendNames() {
		betaVal = MergeBetaHeaderAppend(betaVal, extra)
	}
	betaVal = mergeBodyAnthropicBetasIntoHeader(betaVal, body.AnthropicBeta, c.Provider)
	betaVal = c.mergeSessionLatchedBetas(betaVal, pol)
	return betaVal
}

func (c *Client) applyMessagesRequestHeaders(req *http.Request, betaVal string) {
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
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
}

// PostMessages starts a non-streaming JSON request (executeNonStreamingRequest / anthropic.beta.messages.create stream:false).
func (c *Client) PostMessages(ctx context.Context, body MessagesStreamBody, pol Policy) (*http.Response, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	raw, err := c.marshalMessagesJSON(body, false, pol)
	if err != nil {
		return nil, err
	}
	reqCtx := ctx
	if c.LLMDeriveContext != nil {
		reqCtx = c.LLMDeriveContext(ctx, body.Model, false)
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.messagesURL(body), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	payload := raw
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	betaVal := c.messagesBetaHeader(body, pol)
	c.applyMessagesRequestHeaders(req, betaVal)
	start := time.Now()
	if c.LLMSpanStart != nil {
		c.LLMSpanStart(reqCtx, LLMSpanInfo{
			Model:       body.Model,
			Streaming:   false,
			BodyBytes:   len(raw),
			EffortToken: pol.EffortToken,
			RequestID:   pol.RequestID,
			QuerySource: pol.QuerySource,
		})
	}
	resp, err := DoRequest(reqCtx, c.effectiveTransport(), req, pol)
	if c.LLMSpanEnd != nil {
		sc := 0
		if resp != nil {
			sc = resp.StatusCode
		}
		c.LLMSpanEnd(reqCtx, LLMSpanEndInfo{Streaming: false, StatusCode: sc, Err: err, Duration: time.Since(start)})
	}
	return resp, err
}

// PostMessagesReadAssistantNonStream posts stream:false and decodes the JSON message (text blocks + usage).
func (c *Client) PostMessagesReadAssistantNonStream(ctx context.Context, body MessagesStreamBody, pol Policy, _ ...ReadAssistantOption) (string, UsageDelta, error) {
	nsCtx := ctx
	if d := NonStreamingFallbackTimeout(); d > 0 {
		var cancel context.CancelFunc
		nsCtx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}
	resp, err := c.PostMessages(nsCtx, body, pol)
	if err != nil {
		return "", UsageDelta{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", UsageDelta{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", UsageDelta{}, fmt.Errorf("messages non-stream: status %d: %s", resp.StatusCode, string(b))
	}
	text, u, err := DecodeNonStreamingMessageResponse(b)
	if err != nil {
		return "", UsageDelta{}, err
	}
	if c.OnStreamUsage != nil {
		c.OnStreamUsage(u)
	}
	return text, u, nil
}

// PostMessagesStream starts a streaming request. Caller must close resp.Body.
func (c *Client) PostMessagesStream(ctx context.Context, body MessagesStreamBody, pol Policy) (*http.Response, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	raw, err := c.marshalMessagesStreamJSON(body, pol)
	if err != nil {
		return nil, err
	}
	reqCtx := ctx
	if c.LLMDeriveContext != nil {
		reqCtx = c.LLMDeriveContext(ctx, body.Model, true)
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.messagesURL(body), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	payload := raw
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	betaVal := c.messagesBetaHeader(body, pol)
	c.applyMessagesRequestHeaders(req, betaVal)
	start := time.Now()
	if c.LLMSpanStart != nil {
		c.LLMSpanStart(reqCtx, LLMSpanInfo{
			Model:       body.Model,
			Streaming:   true,
			BodyBytes:   len(raw),
			EffortToken: pol.EffortToken,
			RequestID:   pol.RequestID,
			QuerySource: pol.QuerySource,
		})
	}
	resp, err := DoRequest(reqCtx, c.effectiveTransport(), req, pol)
	if c.LLMSpanEnd != nil {
		sc := 0
		if resp != nil {
			sc = resp.StatusCode
		}
		c.LLMSpanEnd(reqCtx, LLMSpanEndInfo{Streaming: true, StatusCode: sc, Err: err, Duration: time.Since(start)})
	}
	return resp, err
}

// PostMessagesStreamReadAssistantWithNonStreamFallback tries streaming first; on stream read failure with RABBIT_CODE_NONSTREAM_FALLBACK_ON_STREAM_ERROR, retries non-streaming (executeNonStreamingRequest).
func (c *Client) PostMessagesStreamReadAssistantWithNonStreamFallback(ctx context.Context, body MessagesStreamBody, pol Policy, extra ...ReadAssistantOption) (string, UsageDelta, error) {
	if !features.NonStreamFallbackOnStreamError() {
		return c.PostMessagesStreamReadAssistant(ctx, body, pol, extra...)
	}
	resp, err := c.PostMessagesStream(ctx, body, pol)
	if err != nil {
		return "", UsageDelta{}, err
	}
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
	ropts = append(ropts, extra...)
	text, u, err := ReadAssistantStream(ctx, resp.Body, ropts...)
	resp.Body.Close()
	if err == nil {
		if c.OnStreamUsage != nil {
			c.OnStreamUsage(u)
		}
		return text, u, nil
	}
	if ctx.Err() != nil {
		return text, u, err
	}
	bodyNS := AdjustMessagesStreamBodyForNonStreaming(body)
	return c.PostMessagesReadAssistantNonStream(ctx, bodyNS, pol, extra...)
}

// PostMessagesStreamReadAssistant posts, reads the SSE body to completion, closes the response, then
// calls OnStreamUsage when set (P4.1.3 / P4.4.1).
func (c *Client) PostMessagesStreamReadAssistant(ctx context.Context, body MessagesStreamBody, pol Policy, extra ...ReadAssistantOption) (string, UsageDelta, error) {
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
	ropts = append(ropts, extra...)
	text, u, err := ReadAssistantStream(ctx, resp.Body, ropts...)
	if err != nil {
		return text, u, err
	}
	if c.OnStreamUsage != nil {
		c.OnStreamUsage(u)
	}
	return text, u, nil
}

// PostMessagesStreamReadAssistantTurn posts, reads SSE, and returns text + tool_use blocks + stop_reason (Phase 5 query loop).
func (c *Client) PostMessagesStreamReadAssistantTurn(ctx context.Context, body MessagesStreamBody, pol Policy, extra ...ReadAssistantOption) (AssistantStreamTurn, UsageDelta, error) {
	resp, err := c.PostMessagesStream(ctx, body, pol)
	if err != nil {
		return AssistantStreamTurn{}, UsageDelta{}, err
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
	ropts = append(ropts, extra...)
	turn, u, err := ReadAssistantStreamTurn(ctx, resp.Body, ropts...)
	if err != nil {
		return turn, u, err
	}
	if c.OnStreamUsage != nil {
		c.OnStreamUsage(u)
	}
	return turn, u, nil
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
