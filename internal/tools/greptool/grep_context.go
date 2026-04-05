package greptool

import "context"

type grepCtxKey struct{}

// GrepContext mirrors GrepTool.ts call-time inputs (ignore globs, file-read deny patterns, optional DenyRead filter).
type GrepContext struct {
	IgnoreGlobs                []string
	FileReadDenyPatternsByRoot map[string][]string
	DenyRead                   func(absPath string) bool
}

// WithGrepContext attaches *GrepContext for Grep.Run.
func WithGrepContext(ctx context.Context, gc *GrepContext) context.Context {
	if gc == nil {
		return ctx
	}
	return context.WithValue(ctx, grepCtxKey{}, gc)
}

// GrepContextFrom returns *GrepContext or nil.
func GrepContextFrom(ctx context.Context) *GrepContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(grepCtxKey{}).(*GrepContext)
	return v
}
