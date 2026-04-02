package querydeps

import "errors"

// ErrNilAnthropicClient is returned when AnthropicAssistant has a nil Client.
var ErrNilAnthropicClient = errors.New("querydeps: nil anthropic client")
