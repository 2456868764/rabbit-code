package app

import (
	"sync"
)

// CleanupRegistry is the registerCleanup / gracefulShutdown equivalent: LIFO execution.
type CleanupRegistry struct {
	mu   sync.Mutex
	fns  []func()
	done bool
}

// Register adds a cleanup callback; it runs once, in reverse registration order.
func (r *CleanupRegistry) Register(fn func()) {
	if fn == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return
	}
	r.fns = append(r.fns, fn)
}

// Run executes all cleanups in reverse order. Safe to call multiple times (no-op after first).
func (r *CleanupRegistry) Run() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return
	}
	r.done = true
	for i := len(r.fns) - 1; i >= 0; i-- {
		r.fns[i]()
	}
	r.fns = nil
}
