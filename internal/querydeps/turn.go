package querydeps

import (
	"context"
	"encoding/json"
)

// ToolUseCall is one tool_use block in an assistant message (Messages API shape).
type ToolUseCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// TurnResult is one assistant model response (text + optional tool uses).
type TurnResult struct {
	Text      string
	ToolUses  []ToolUseCall
}

// TurnAssistant performs one assistant turn and may return tool calls (P5.1.2 / AC5-3).
type TurnAssistant interface {
	AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (TurnResult, error)
}

// StreamAsTurnAssistant wraps StreamAssistant as text-only turns (no tool_uses).
func StreamAsTurnAssistant(s StreamAssistant) TurnAssistant {
	if s == nil {
		return NoopTurnAssistant{}
	}
	return streamAsTurn{s: s}
}

type streamAsTurn struct {
	s StreamAssistant
}

func (a streamAsTurn) AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (TurnResult, error) {
	text, err := a.s.StreamAssistant(ctx, model, maxTokens, messagesJSON)
	if err != nil {
		return TurnResult{}, err
	}
	return TurnResult{Text: text}, nil
}

// NoopTurnAssistant returns an empty turn (stops RunTurnLoop immediately when no prior work).
type NoopTurnAssistant struct{}

func (NoopTurnAssistant) AssistantTurn(context.Context, string, int, []byte) (TurnResult, error) {
	return TurnResult{}, nil
}
