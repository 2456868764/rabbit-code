package query

import (
	"context"
	"errors"
	"sync"
)

// StreamAssistantFunc adapts a function to StreamAssistant (tests and small harnesses).
type StreamAssistantFunc func(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error)

// StreamAssistant implements StreamAssistant.
func (f StreamAssistantFunc) StreamAssistant(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
	return f(ctx, model, maxTokens, messagesJSON)
}

// SequenceAssistant returns Replies[i] on the i-th StreamAssistant call (multi-turn tests / AC5-3 prep).
type SequenceAssistant struct {
	Replies []string
	mu      sync.Mutex
	i       int
}

// StreamAssistant implements StreamAssistant.
func (s *SequenceAssistant) StreamAssistant(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
	_ = ctx
	_ = model
	_ = maxTokens
	_ = messagesJSON
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.i >= len(s.Replies) {
		return "", ErrSequenceExhausted
	}
	r := s.Replies[s.i]
	s.i++
	return r, nil
}

// SequenceTurnAssistant returns Turns[i] on the i-th AssistantTurn call (mock multi-round + tools / AC5-3).
type SequenceTurnAssistant struct {
	Turns []TurnResult
	mu    sync.Mutex
	i     int
}

// AssistantTurn implements TurnAssistant.
func (s *SequenceTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (TurnResult, error) {
	_ = ctx
	_ = model
	_ = maxTokens
	_ = messagesJSON
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.i >= len(s.Turns) {
		return TurnResult{}, ErrSequenceExhausted
	}
	r := s.Turns[s.i]
	s.i++
	return r, nil
}

// ErrSequenceExhausted is returned when Replies is consumed.
var ErrSequenceExhausted = errors.New("query: sequence assistant exhausted")
