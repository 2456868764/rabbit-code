package engine

import "errors"

// ErrTokenBudgetExceeded is returned when RABBIT_CODE_TOKEN_BUDGET is on and resolved Submit text
// exceeds RABBIT_CODE_TOKEN_BUDGET_MAX_INPUT_BYTES (UTF-8 byte length).
var ErrTokenBudgetExceeded = errors.New("engine: input exceeds token budget limit")
