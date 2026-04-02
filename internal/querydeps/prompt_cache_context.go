package querydeps

import "context"

type ctxKeyOnPromptCacheBreak struct{}

// ContextWithOnPromptCacheBreak stores fn for AnthropicAssistant stream readers (P5.F.9).
// Safe for concurrent Submits: each RunTurnLoop uses its own context chain.
func ContextWithOnPromptCacheBreak(ctx context.Context, fn func()) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyOnPromptCacheBreak{}, fn)
}

// OnPromptCacheBreakFromContext returns the callback from ContextWithOnPromptCacheBreak.
func OnPromptCacheBreakFromContext(ctx context.Context) (func(), bool) {
	v, ok := ctx.Value(ctxKeyOnPromptCacheBreak{}).(func())
	return v, ok
}
