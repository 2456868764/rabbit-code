package anthropic

import "errors"

// ErrNilAnthropicClient is returned when AnthropicAssistant has a nil Client.
var ErrNilAnthropicClient = errors.New("anthropic: nil anthropic client")
