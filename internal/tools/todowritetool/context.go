package todowritetool

import (
	"context"
	"strings"
	"sync"

	"slices"
)

type runCtxKey struct{}

// RunContext carries session todo storage (TodoWriteTool.ts ToolUseContext appState.todos analogue).
type RunContext struct {
	SessionID      string
	AgentID        string
	NonInteractive bool
	Store          *Store
}

// Store is a keyed todo list map (thread-safe). Keys match TS appState.todos (agentId ?? sessionId).
type Store struct {
	mu sync.RWMutex
	m  map[string][]TodoItem
}

// NewStore returns an empty todo store.
func NewStore() *Store {
	return &Store{m: make(map[string][]TodoItem)}
}

// Get returns a copy of todos for key, or nil if missing.
func (s *Store) Get(key string) []TodoItem {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	if !ok {
		return nil
	}
	return slices.Clone(v)
}

// Set replaces todos for key (stored copy).
func (s *Store) Set(key string, todos []TodoItem) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = make(map[string][]TodoItem)
	}
	s.m[key] = slices.Clone(todos)
}

// TodoKey returns context.agentId ?? getSessionId() (TodoWriteTool.ts). AgentID and SessionID
// should mirror toolUseContext.agentId and bootstrap getSessionId() when wired; empty key shares one bucket.
func TodoKey(rc *RunContext) string {
	if rc == nil {
		return ""
	}
	if a := strings.TrimSpace(rc.AgentID); a != "" {
		return a
	}
	return strings.TrimSpace(rc.SessionID)
}

// WithRunContext attaches *RunContext for TodoWrite.Run.
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
