package querydeps

import (
	"context"
	"errors"
	"sync"
)

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

// ErrSequenceExhausted is returned when Replies is consumed.
var ErrSequenceExhausted = errors.New("querydeps: sequence assistant exhausted")
