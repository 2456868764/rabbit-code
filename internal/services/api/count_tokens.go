package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BetaTokenCounting is the anthropic-beta feature flag for POST /v1/messages/count_tokens (docs token counting API).
const BetaTokenCounting = "token-counting-2024-11-01"

// ErrCountTokensUnsupported means the configured Provider does not expose the Anthropic count_tokens HTTP API.
var ErrCountTokensUnsupported = errors.New("anthropic: count_tokens is only supported for ProviderAnthropic")

type countTokensRequest struct {
	Model    string          `json:"model"`
	Messages json.RawMessage `json:"messages"`
}

type countTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}

func (c *Client) countTokensURL() string {
	base := strings.TrimRight(c.BaseURL, "/")
	if c.Provider != ProviderAnthropic {
		return ""
	}
	return base + "/v1/messages/count_tokens"
}

// CountMessagesInputTokens calls POST /v1/messages/count_tokens (1P Anthropic only). messages must be a JSON array.
func (c *Client) CountMessagesInputTokens(ctx context.Context, model string, messages json.RawMessage, pol Policy) (int, error) {
	if c == nil {
		return 0, errors.New("anthropic: nil client")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Provider != ProviderAnthropic {
		return 0, ErrCountTokensUnsupported
	}
	url := c.countTokensURL()
	if url == "" {
		return 0, ErrCountTokensUnsupported
	}
	body := countTokensRequest{Model: model, Messages: messages}
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	payload := raw
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	beta := MergeBetaHeaderAppend(c.BetaHeader, BetaTokenCounting)
	if beta != "" {
		req.Header.Set("anthropic-beta", beta)
	}
	if sid := strings.TrimSpace(c.SessionID); sid != "" {
		req.Header.Set("X-Claude-Code-Session-Id", sid)
	}
	for k, vv := range MergedCustomHeadersFromEnv() {
		if len(vv) > 0 {
			req.Header.Set(k, vv[0])
		}
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", UserAgent())
	}
	for k, v := range c.ExtraHeaders {
		req.Header[k] = append([]string(nil), v...)
	}
	if pol.MaxAttempts == 0 {
		pol = DefaultPolicy()
	}
	resp, err := DoRequest(ctx, c.effectiveTransport(), req, pol)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("anthropic count_tokens: HTTP %d: %s", resp.StatusCode, bytesTrimPreview(b, 512))
	}
	var out countTokensResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return 0, fmt.Errorf("anthropic count_tokens: decode: %w", err)
	}
	return out.InputTokens, nil
}

func bytesTrimPreview(b []byte, max int) string {
	s := string(b)
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
