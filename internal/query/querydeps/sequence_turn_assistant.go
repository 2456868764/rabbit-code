package querydeps

import (
	"context"
	"sync"
)

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
