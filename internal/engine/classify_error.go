package engine

import (
	"errors"

	"github.com/2456868764/rabbit-code/internal/anthropic"
)

// classifyAnthropicError returns API error kind string and whether compact/trim recovery is suggested (P5.1.3).
func classifyAnthropicError(err error) (kind string, recoverableCompact bool) {
	if err == nil {
		return "", false
	}
	var api *anthropic.APIError
	if errors.As(err, &api) {
		k := string(api.Kind)
		rec := api.Kind == anthropic.KindPromptTooLong || api.Kind == anthropic.KindMaxOutputTokens
		return k, rec
	}
	return "", false
}
