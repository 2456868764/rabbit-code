package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// CompactSummaryStreamResult holds raw assistant text (PTL / hook parity) and formatted summary (UI).
type CompactSummaryStreamResult struct {
	Formatted string
	Raw       string
}

// StreamCompactSummary runs streamCompactSummary (fork when configured, else streaming + PTL truncate + optional streaming retries).
func (a *AnthropicAssistant) StreamCompactSummary(ctx context.Context, transcriptJSON []byte, customInstructions string) (string, error) {
	r, err := a.StreamCompactSummaryDetailed(ctx, transcriptJSON, customInstructions)
	if err != nil {
		return "", err
	}
	return r.Formatted, nil
}

// StreamCompactSummaryDetailed mirrors compact.ts outcomes needed for post_compact hooks and next-transcript builders (raw vs formatted).
func (a *AnthropicAssistant) StreamCompactSummaryDetailed(ctx context.Context, transcriptJSON []byte, customInstructions string) (CompactSummaryStreamResult, error) {
	var zero CompactSummaryStreamResult
	if a == nil || a.Client == nil {
		return zero, ErrNilAnthropicClient
	}
	stopKeepAlive := a.startCompactKeepAlive(ctx)
	defer stopKeepAlive()
	rawIn := bytes.TrimSpace(transcriptJSON)
	if len(rawIn) == 0 || string(rawIn) == "null" || string(rawIn) == "[]" {
		return zero, errors.New(compact.ErrorMessageNotEnoughMessages)
	}
	model := strings.TrimSpace(a.DefaultModel)
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	maxTok := compact.EffectiveCompactSummaryMaxTokens(a.DefaultMaxTokens)
	pol := a.Policy
	if pol.MaxAttempts == 0 {
		pol = DefaultPolicy()
	}

	customInst := customInstructions

	if features.CompactCachePrefixEnabled() && a.ForkCompactSummary != nil {
		sumUser, err := compact.CreateUserTextMessageJSON(compact.GetCompactPrompt(customInst))
		if err == nil {
			if raw, err := a.ForkCompactSummary(ctx, sumUser, transcriptJSON); err == nil && strings.TrimSpace(raw) != "" {
				if validCompactAssistantText(raw) {
					return CompactSummaryStreamResult{
						Formatted: compact.FormatCompactSummary(raw),
						Raw:       raw,
					}, nil
				}
			}
		}
	}

	messages := append([]byte(nil), transcriptJSON...)
	for ptlAttempts := 0; ; {
		maxStreamAttempts := 1
		if features.CompactStreamingRetryEnabled() {
			maxStreamAttempts = compact.MaxCompactStreamingRetries
		}
		var text string
		var err error
		for attempt := 1; attempt <= maxStreamAttempts; attempt++ {
			text, err = a.streamCompactSummaryOnce(ctx, model, maxTok, messages, customInst, pol)
			if err != nil {
				return zero, err
			}
			if strings.TrimSpace(text) != "" {
				break
			}
			if attempt == maxStreamAttempts {
				return zero, errors.New(compact.ErrorMessageIncompleteResponse)
			}
		}

		if !compact.CompactSummaryLooksLikePromptTooLong(text) {
			if strings.TrimSpace(text) == "" {
				return zero, errors.New(compact.ErrorMessageNoCompactSummary)
			}
			if compact.StartsWithAPIErrorPrefix(text) {
				return zero, fmt.Errorf("%s", text)
			}
			if strings.TrimSpace(text) == compact.ErrorMessageUserAbort || strings.Contains(text, "Request was aborted") {
				return zero, errors.New(compact.ErrorMessageUserAbort)
			}
			return CompactSummaryStreamResult{
				Formatted: compact.FormatCompactSummary(text),
				Raw:       text,
			}, nil
		}
		ptlAttempts++
		if ptlAttempts > compact.MaxPTLRetries {
			return zero, errors.New(compact.ErrorMessagePromptTooLong)
		}
		asst := assistantTextMessageJSONRaw(text)
		next, ok := compact.TruncateHeadForPTLRetryTranscriptJSON(messages, asst)
		if !ok {
			return zero, errors.New(compact.ErrorMessagePromptTooLong)
		}
		messages = next
	}
}

// StreamPartialCompactSummaryDetailed mirrors compact.ts partialCompactConversation streaming summary path
// (pivot + direction slice via BuildPartialCompactStreamRequestMessagesJSON; PTL retries truncate the built request).
func (a *AnthropicAssistant) StreamPartialCompactSummaryDetailed(ctx context.Context, fullTranscriptJSON []byte, pivot int, direction compact.PartialCompactDirection, customInstructions string) (CompactSummaryStreamResult, error) {
	var zero CompactSummaryStreamResult
	if a == nil || a.Client == nil {
		return zero, ErrNilAnthropicClient
	}
	stopKeepAlive := a.startCompactKeepAlive(ctx)
	defer stopKeepAlive()
	rawIn := bytes.TrimSpace(fullTranscriptJSON)
	if len(rawIn) == 0 || string(rawIn) == "null" || string(rawIn) == "[]" {
		return zero, errors.New(compact.ErrorMessageNotEnoughMessages)
	}
	if _, err := compact.SelectPartialCompactAPIMessagesTranscriptJSON(fullTranscriptJSON, pivot, direction); err != nil {
		return zero, err
	}
	model := strings.TrimSpace(a.DefaultModel)
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	maxTok := compact.EffectiveCompactSummaryMaxTokens(a.DefaultMaxTokens)
	pol := a.Policy
	if pol.MaxAttempts == 0 {
		pol = DefaultPolicy()
	}
	customInst := customInstructions

	requestMsgs, err := compact.BuildPartialCompactStreamRequestMessagesJSON(fullTranscriptJSON, pivot, direction, customInst)
	if err != nil {
		return zero, err
	}

	if features.CompactCachePrefixEnabled() && a.ForkPartialCompactSummary != nil {
		if raw, err := a.ForkPartialCompactSummary(ctx, requestMsgs); err == nil && strings.TrimSpace(raw) != "" {
			if validCompactAssistantText(raw) {
				return CompactSummaryStreamResult{
					Formatted: compact.FormatCompactSummary(raw),
					Raw:       raw,
				}, nil
			}
		}
	}

	for ptlAttempts := 0; ; {
		maxStreamAttempts := 1
		if features.CompactStreamingRetryEnabled() {
			maxStreamAttempts = compact.MaxCompactStreamingRetries
		}
		var text string
		for attempt := 1; attempt <= maxStreamAttempts; attempt++ {
			text, err = a.streamCompactSummaryFromMessages(ctx, model, maxTok, requestMsgs, pol)
			if err != nil {
				return zero, err
			}
			if strings.TrimSpace(text) != "" {
				break
			}
			if attempt == maxStreamAttempts {
				return zero, errors.New(compact.ErrorMessageIncompleteResponse)
			}
		}

		if !compact.CompactSummaryLooksLikePromptTooLong(text) {
			if strings.TrimSpace(text) == "" {
				return zero, errors.New(compact.ErrorMessageNoCompactSummary)
			}
			if compact.StartsWithAPIErrorPrefix(text) {
				return zero, fmt.Errorf("%s", text)
			}
			if strings.TrimSpace(text) == compact.ErrorMessageUserAbort || strings.Contains(text, "Request was aborted") {
				return zero, errors.New(compact.ErrorMessageUserAbort)
			}
			return CompactSummaryStreamResult{
				Formatted: compact.FormatCompactSummary(text),
				Raw:       text,
			}, nil
		}
		ptlAttempts++
		if ptlAttempts > compact.MaxPTLRetries {
			return zero, errors.New(compact.ErrorMessagePromptTooLong)
		}
		asst := assistantTextMessageJSONRaw(text)
		next, ok := compact.TruncateHeadForPTLRetryTranscriptJSON(requestMsgs, asst)
		if !ok {
			return zero, errors.New(compact.ErrorMessagePromptTooLong)
		}
		requestMsgs = next
	}
}

// startCompactKeepAlive mirrors compact.ts streamCompactSummary setInterval(sessionActivity) when callback + env gate are set.
func (a *AnthropicAssistant) startCompactKeepAlive(ctx context.Context) func() {
	if a == nil || a.SessionActivityPing == nil || !features.RemoteSendKeepalivesEnabled() {
		return func() {}
	}
	d := a.CompactKeepAliveInterval
	if d <= 0 {
		d = 30 * time.Second
	}
	subCtx, cancel := context.WithCancel(ctx)
	t := time.NewTicker(d)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-subCtx.Done():
				return
			case <-t.C:
				if ping := a.SessionActivityPing; ping != nil {
					ping(ctx)
				}
			}
		}
	}()
	return cancel
}

func validCompactAssistantText(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" || compact.CompactSummaryLooksLikePromptTooLong(s) {
		return false
	}
	if compact.StartsWithAPIErrorPrefix(s) {
		return false
	}
	if s == compact.ErrorMessageUserAbort || strings.Contains(s, "Request was aborted") {
		return false
	}
	return true
}

func assistantTextMessageJSONRaw(text string) []byte {
	m := map[string]interface{}{
		"role": "assistant",
		"content": []interface{}{
			map[string]string{"type": "text", "text": text},
		},
	}
	b, _ := json.Marshal(m)
	return b
}

func (a *AnthropicAssistant) streamCompactSummaryFromMessages(ctx context.Context, model string, maxTokens int, messagesJSON []byte, pol Policy) (string, error) {
	body := a.compactSummaryStreamBody(model, maxTokens, messagesJSON)
	text, _, err := a.Client.PostMessagesStreamReadAssistant(ctx, body, pol, a.readOpts(ctx)...)
	return text, err
}

func (a *AnthropicAssistant) streamCompactSummaryOnce(ctx context.Context, model string, maxTokens int, transcriptJSON []byte, customInstructions string, pol Policy) (string, error) {
	prompt := compact.GetCompactPrompt(customInstructions)
	msgs, err := compact.BuildCompactStreamRequestMessagesJSON(transcriptJSON, compact.AfterCompactBoundaryOptions{}, prompt)
	if err != nil {
		return "", err
	}
	return a.streamCompactSummaryFromMessages(ctx, model, maxTokens, msgs, pol)
}

func (a *AnthropicAssistant) compactSummaryStreamBody(model string, maxTokens int, messagesJSON []byte) MessagesStreamBody {
	body := MessagesStreamBody{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  json.RawMessage(messagesJSON),
	}
	if b, err := json.Marshal(compact.CompactSummarySystemPromptEnglish); err == nil {
		body.System = b
	}
	if tools, err := a.compactToolsJSONResolved(); err == nil && len(bytes.TrimSpace(tools)) > 0 {
		body.Tools = tools
	}
	for _, beta := range a.compactBetas() {
		body.AnthropicBeta = AppendBetaUnique(body.AnthropicBeta, beta)
	}
	if features.CachedMicrocompactEnabled() {
		body.AnthropicBeta = AppendBetaUnique(body.AnthropicBeta, BetaCachedMicrocompactBody)
	}
	if a != nil {
		a.attachAPIContextManagement(model, &body)
	}
	return body
}

func (a *AnthropicAssistant) compactToolsJSONResolved() (json.RawMessage, error) {
	if a != nil && len(bytes.TrimSpace(a.CompactToolsJSON)) > 0 {
		return a.CompactToolsJSON, nil
	}
	return compact.DefaultCompactStreamingToolsJSON(features.CompactStreamingToolSearchEnabled())
}

func (a *AnthropicAssistant) compactBetas() []string {
	if a == nil || len(a.CompactStreamExtraBetas) == 0 {
		return nil
	}
	return append([]string(nil), a.CompactStreamExtraBetas...)
}
