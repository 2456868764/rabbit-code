package websearchtool

import "context"

type runCtxKey struct{}

// RunContext carries optional WebSearch.Run overrides (ExecuteSearch mirrors streaming API path).
type RunContext struct {
	ExecuteSearch func(ctx context.Context, in Input) (results []any, err error)
	// OnWebSearchProgress optional; mirrors WebSearchTool.call onProgress (query_update, search_results_received).
	OnWebSearchProgress func(ev WebSearchProgress)
}

// WithRunContext attaches *RunContext for WebSearch.Run.
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
