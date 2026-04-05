package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

// MaxNonStreamingTokens caps max_tokens for non-streaming fallback (claude.ts MAX_NON_STREAMING_TOKENS).
const MaxNonStreamingTokens = 64_000

// AdjustMessagesStreamBodyForNonStreaming caps max_tokens and thinking budget (adjustParamsForNonStreaming).
func AdjustMessagesStreamBodyForNonStreaming(body MessagesStreamBody) MessagesStreamBody {
	capped := body.MaxTokens
	if capped > MaxNonStreamingTokens {
		capped = MaxNonStreamingTokens
	}
	body.MaxTokens = capped
	if len(body.Thinking) == 0 || capped < 2 {
		return body
	}
	var th map[string]any
	if err := json.Unmarshal(body.Thinking, &th); err != nil {
		return body
	}
	if typ, _ := th["type"].(string); typ != "enabled" {
		return body
	}
	bt, ok := th["budget_tokens"].(float64)
	if !ok {
		return body
	}
	maxBudget := capped - 1
	if maxBudget < 1 {
		maxBudget = 1
	}
	bti := int(bt)
	if bti > maxBudget {
		th["budget_tokens"] = float64(maxBudget)
		b, err := json.Marshal(th)
		if err == nil {
			body.Thinking = b
		}
	}
	return body
}

// NonStreamingFallbackTimeout returns per-request timeout for non-streaming fallback (getNonstreamingFallbackTimeoutMs).
func NonStreamingFallbackTimeout() time.Duration {
	if ms, ok := parseEnvIntMS("API_TIMEOUT_MS"); ok && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	if ms, ok := parseEnvIntMS("RABBIT_CODE_NONSTREAMING_FALLBACK_TIMEOUT_MS"); ok && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	if features.ClaudeCodeRemote() {
		return 120 * time.Second
	}
	return 300 * time.Second
}

func parseEnvIntMS(key string) (int, bool) {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// nonStreamMessageResponse is a minimal shape for POST /v1/messages JSON (stream:false).
type nonStreamMessageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens              int64 `json:"input_tokens"`
		CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		OutputTokens             int64 `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// DecodeNonStreamingMessageResponse extracts assistant text (text blocks) and usage from a Messages API JSON body.
func DecodeNonStreamingMessageResponse(data []byte) (string, UsageDelta, error) {
	var resp nonStreamMessageResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", UsageDelta{}, err
	}
	if resp.Error != nil && (resp.Error.Message != "" || resp.Error.Type != "") {
		return "", UsageDelta{}, fmt.Errorf("messages api error: %s: %s", resp.Error.Type, resp.Error.Message)
	}
	var b strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	u := UsageDelta{
		InputTokens:              resp.Usage.InputTokens,
		CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
		CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		OutputTokens:             resp.Usage.OutputTokens,
	}
	return b.String(), u, nil
}

// ReadNonStreamingMessageBody parses a successful non-streaming HTTP response body into text + usage.
func ReadNonStreamingMessageBody(r io.Reader) (string, UsageDelta, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", UsageDelta{}, err
	}
	return DecodeNonStreamingMessageResponse(data)
}
