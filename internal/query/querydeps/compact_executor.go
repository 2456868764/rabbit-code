package querydeps

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// StreamingCompactExecutor returns engine.CompactExecutor using default options (customInstructions only).
func StreamingCompactExecutor(a *AnthropicAssistant, customInstructions string) func(context.Context, compact.RunPhase, []byte) (summary string, nextTranscriptJSON []byte, err error) {
	return StreamingCompactExecutorWithConfig(a, StreamingCompactExecutorConfig{
		CustomInstructions: customInstructions,
	})
}

// StreamingCompactExecutorConfig mirrors optional compact.ts compactConversation hooks and post-compact transcript assembly.
type StreamingCompactExecutorConfig struct {
	CustomInstructions        string
	SuppressFollowUpQuestions bool
	TranscriptPath            string
	LastPreCompactMessageUUID string
	ReturnNextTranscript      bool
	PreCompactHook            func(ctx context.Context, autoCompact bool) (hookInstructions string, userDisplay string, err error)
	PostCompactHook           func(ctx context.Context, autoCompact bool, rawSummary string) (userDisplay string, err error)
	SessionStartHook          func(ctx context.Context) (hookMessages []json.RawMessage, err error)
	// PostCompactAttachmentsJSON optional; when ReturnNextTranscript, merged into next transcript before SessionStartHook messages (compact.ts postCompactFileAttachments order: host supplies JSON).
	PostCompactAttachmentsJSON func(ctx context.Context, transcriptBeforeCompact []byte, rawAssistantSummary string) ([]json.RawMessage, error)
}

// StreamingCompactExecutorWithConfig runs StreamCompactSummaryDetailed with MergeHookInstructions, optional hooks, and optional BuildDefaultPostCompactTranscriptJSON next payload.
// Auto vs manual uses compact.ExecutorSuggestMetaFromContext (set by engine around CompactExecutor).
func StreamingCompactExecutorWithConfig(a *AnthropicAssistant, cfg StreamingCompactExecutorConfig) func(context.Context, compact.RunPhase, []byte) (summary string, nextTranscriptJSON []byte, err error) {
	return func(ctx context.Context, _ compact.RunPhase, transcriptJSON []byte) (string, []byte, error) {
		if a == nil || a.Client == nil {
			return "", nil, ErrNilAnthropicClient
		}
		auto := false
		if m, ok := compact.ExecutorSuggestMetaFromContext(ctx); ok {
			auto = m.AutoCompact
		}
		customInst := cfg.CustomInstructions
		var displayParts []string
		if cfg.PreCompactHook != nil {
			hi, ud, err := cfg.PreCompactHook(ctx, auto)
			if err != nil {
				return "", nil, err
			}
			customInst = compact.MergeHookInstructions(customInst, hi)
			if s := strings.TrimSpace(ud); s != "" {
				displayParts = append(displayParts, s)
			}
		}
		res, err := a.StreamCompactSummaryDetailed(ctx, transcriptJSON, customInst)
		if err != nil {
			return "", nil, err
		}
		if cfg.PostCompactHook != nil {
			ud, err := cfg.PostCompactHook(ctx, auto, res.Raw)
			if err != nil {
				return "", nil, err
			}
			if s := strings.TrimSpace(ud); s != "" {
				displayParts = append(displayParts, s)
			}
		}
		var hookMsgs []json.RawMessage
		if cfg.SessionStartHook != nil {
			var hErr error
			hookMsgs, hErr = cfg.SessionStartHook(ctx)
			if hErr != nil {
				return "", nil, hErr
			}
		}
		outSummary := res.Formatted
		if len(displayParts) > 0 {
			outSummary = strings.Join(displayParts, "\n") + "\n\n" + outSummary
		}
		if !cfg.ReturnNextTranscript {
			return outSummary, nil, nil
		}
		var extraAtt []json.RawMessage
		if cfg.PostCompactAttachmentsJSON != nil {
			var aErr error
			extraAtt, aErr = cfg.PostCompactAttachmentsJSON(ctx, transcriptJSON, res.Raw)
			if aErr != nil {
				return "", nil, aErr
			}
		}
		next, err := compact.BuildDefaultPostCompactTranscriptJSON(transcriptJSON, res.Raw, compact.PostCompactTranscriptOptions{
			AutoCompact:               auto,
			SuppressFollowUpQuestions: cfg.SuppressFollowUpQuestions,
			TranscriptPath:            cfg.TranscriptPath,
			LastPreCompactMessageUUID: cfg.LastPreCompactMessageUUID,
			ExtraAttachmentsJSON:      extraAtt,
			HookResultMessagesJSON:    hookMsgs,
		})
		if err != nil {
			return outSummary, nil, err
		}
		return outSummary, next, nil
	}
}

// StreamingPartialCompactExecutorWithConfig mirrors partialCompactConversation + StreamingCompactExecutorWithConfig:
// summary stream uses GetPartialCompactPrompt; hooks and next-transcript behave like full compact.
func StreamingPartialCompactExecutorWithConfig(a *AnthropicAssistant, pivot int, direction compact.PartialCompactDirection, cfg StreamingCompactExecutorConfig) func(context.Context, compact.RunPhase, []byte) (summary string, nextTranscriptJSON []byte, err error) {
	return func(ctx context.Context, _ compact.RunPhase, transcriptJSON []byte) (string, []byte, error) {
		if a == nil || a.Client == nil {
			return "", nil, ErrNilAnthropicClient
		}
		auto := false
		if m, ok := compact.ExecutorSuggestMetaFromContext(ctx); ok {
			auto = m.AutoCompact
		}
		customInst := cfg.CustomInstructions
		var displayParts []string
		if cfg.PreCompactHook != nil {
			hi, ud, err := cfg.PreCompactHook(ctx, auto)
			if err != nil {
				return "", nil, err
			}
			customInst = compact.MergeHookInstructions(customInst, hi)
			if s := strings.TrimSpace(ud); s != "" {
				displayParts = append(displayParts, s)
			}
		}
		res, err := a.StreamPartialCompactSummaryDetailed(ctx, transcriptJSON, pivot, direction, customInst)
		if err != nil {
			return "", nil, err
		}
		if cfg.PostCompactHook != nil {
			ud, err := cfg.PostCompactHook(ctx, auto, res.Raw)
			if err != nil {
				return "", nil, err
			}
			if s := strings.TrimSpace(ud); s != "" {
				displayParts = append(displayParts, s)
			}
		}
		var hookMsgs []json.RawMessage
		if cfg.SessionStartHook != nil {
			var hErr error
			hookMsgs, hErr = cfg.SessionStartHook(ctx)
			if hErr != nil {
				return "", nil, hErr
			}
		}
		outSummary := res.Formatted
		if len(displayParts) > 0 {
			outSummary = strings.Join(displayParts, "\n") + "\n\n" + outSummary
		}
		if !cfg.ReturnNextTranscript {
			return outSummary, nil, nil
		}
		var extraAtt []json.RawMessage
		if cfg.PostCompactAttachmentsJSON != nil {
			var aErr error
			extraAtt, aErr = cfg.PostCompactAttachmentsJSON(ctx, transcriptJSON, res.Raw)
			if aErr != nil {
				return "", nil, aErr
			}
		}
		next, err := compact.BuildDefaultPostCompactTranscriptJSON(transcriptJSON, res.Raw, compact.PostCompactTranscriptOptions{
			AutoCompact:               auto,
			SuppressFollowUpQuestions: cfg.SuppressFollowUpQuestions,
			TranscriptPath:            cfg.TranscriptPath,
			LastPreCompactMessageUUID: cfg.LastPreCompactMessageUUID,
			ExtraAttachmentsJSON:      extraAtt,
			HookResultMessagesJSON:    hookMsgs,
		})
		if err != nil {
			return outSummary, nil, err
		}
		return outSummary, next, nil
	}
}
