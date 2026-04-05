package filereadtool

import "context"

type runCtxKey struct{}

// RunContext carries optional FileReadTool.ts ToolUseContext analogue for a single Run (permissions, dedup, token API).
type RunContext struct {
	// MainLoopModel for IsPDFSupported (empty = supported).
	MainLoopModel string
	// DenyRead returns true if path must be rejected (always-deny rules).
	DenyRead func(absPath string) bool
	// ReadFileStateKey → entry; nil disables dedup cache reads/writes.
	ReadFileState *ReadFileStateMap
	// CountTokens optional API tokenizer; nil uses heuristic only.
	CountTokens func(text string) (int, error)
	// DisableReadDedup when true skips file_unchanged short-circuit.
	DisableReadDedup bool
	// MaxSizeBytes overrides default file read byte cap when > 0.
	MaxSizeBytes *int
	// MaxTokens overrides default token cap when > 0.
	MaxTokens *int
}

// ReadFileStateMap mirrors readFileState cache (FileReadTool call dedup).
type ReadFileStateMap struct {
	m map[string]ReadFileStateEntry
}

// ReadFileStateEntry mirrors TS readFileState value shape used for Read dedup.
type ReadFileStateEntry struct {
	Content       string
	Timestamp     int64
	Offset        *int
	Limit         *int
	IsPartialView bool
}

func NewReadFileStateMap() *ReadFileStateMap {
	return &ReadFileStateMap{m: make(map[string]ReadFileStateEntry)}
}

func (s *ReadFileStateMap) Get(key string) (ReadFileStateEntry, bool) {
	if s == nil || s.m == nil {
		return ReadFileStateEntry{}, false
	}
	v, ok := s.m[key]
	return v, ok
}

func (s *ReadFileStateMap) Set(key string, e ReadFileStateEntry) {
	if s == nil || s.m == nil {
		return
	}
	s.m[key] = e
}

// WithRunContext returns ctx that carries *RunContext for FileRead.Run.
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
