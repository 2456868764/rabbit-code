package querydeps

import "context"

// StreamAssistantFunc adapts a function to StreamAssistant (tests and small harnesses).
type StreamAssistantFunc func(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error)

// StreamAssistant implements StreamAssistant.
func (f StreamAssistantFunc) StreamAssistant(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
	return f(ctx, model, maxTokens, messagesJSON)
}
