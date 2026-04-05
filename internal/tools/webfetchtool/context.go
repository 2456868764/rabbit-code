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
	// SkipWebFetchPreflight when non-nil overrides env/features (true = skip domain_info preflight).
	SkipWebFetchPreflight *bool
	// ToolResultsDir when non-empty is the directory for persistBinaryWebFetch (default: user cache rabbit-code/tool-results).
	ToolResultsDir string
	// DomainCheckBaseURL when non-empty overrides DefaultDomainCheckBaseURL for domain_info requests.
	DomainCheckBaseURL string
	// DomainCheckClient optional HTTP client for domain preflight only.
	DomainCheckClient *http.Client
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
