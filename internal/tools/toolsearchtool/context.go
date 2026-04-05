package toolsearchtool

import "context"

type runCtxKey struct{}

// RunContext optional overrides for ToolSearch.call (deferred catalog / pending MCP).
type RunContext struct {
	// FullCatalog when non-empty replaces DefaultCatalog().
	FullCatalog []ToolEntry
	// DeferredToolNames when non-empty replaces DefaultDeferredToolNames().
	DeferredToolNames []string
	// PendingMCPServers mirrors appState.mcp.clients pending names (optional).
	PendingMCPServers []string
}

// WithRunContext attaches *RunContext for ToolSearch.Run.
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
