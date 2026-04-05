package webfetchtool

import (
	"context"
	"net/http"
)

type runCtxKey struct{}

// RunContext optional overrides for WebFetch.Run (HTTP client, secondary prompt application).
type RunContext struct {
	HTTPClient *http.Client
	// ApplyPrompt when non-nil replaces the headless fallback (Haiku / queryHaiku upstream).
	ApplyPrompt func(ctx context.Context, markdown, prompt string, preapproved bool) (string, error)
}

// WithRunContext attaches *RunContext for WebFetch.Run.
func WithRunContext(ctx context.Context, rc *RunContext) context.Context {
	if rc == nil {
		return ctx
	}
	return context.WithValue(ctx, runCtxKey{}, rc)
}

// RunContextFrom returns *RunContext or nil.
func RunContextFrom(ctx context.Context) *RunContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(runCtxKey{}).(*RunContext)
	return v
}
