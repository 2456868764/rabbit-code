package query

import "errors"

// ErrNilAnthropicClient is returned when AnthropicAssistant has a nil Client.
var ErrNilAnthropicClient = errors.New("query: nil anthropic client")
