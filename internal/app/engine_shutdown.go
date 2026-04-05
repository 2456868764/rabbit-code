package app

import (
	"context"
	"time"

	"github.com/2456868764/rabbit-code/internal/query/engine"
)

// extractMemoriesDrainTimeout bounds forked extract wait on shutdown (print.ts drainPendingExtraction).
const extractMemoriesDrainTimeout = 5 * time.Minute

// RegisterEngineShutdown registers a cleanup that waits for in-flight EXTRACT_MEMORIES fork work
// before earlier-registered Runtime cleanups run (CleanupRegistry is LIFO; register after Bootstrap
// so drain runs before log close). Uses a fresh timeout context so a cancelled request ctx does not
// skip the drain. No-op if r, Cleanup, or e is nil.
func (r *Runtime) RegisterEngineShutdown(e *engine.Engine) {
	if r == nil || r.Cleanup == nil || e == nil {
		return
	}
	r.Cleanup.Register(func() {
		ctx, cancel := context.WithTimeout(context.Background(), extractMemoriesDrainTimeout)
		defer cancel()
		e.DrainExtractMemories(ctx)
	})
}
