package app

import (
	"context"

	"github.com/2456868764/rabbit-code/internal/query/engine"
)

// WireHeadlessEngineForShutdown creates a minimal engine.Engine (nil Config → stub assistant)
// and registers DrainExtractMemories on Runtime.Close so forked extract can drain before exit
// (H8 / PHASE05_CONTINUATION §3.0 #2; print.ts drainPendingExtraction).
func WireHeadlessEngineForShutdown(rt *Runtime, parent context.Context) {
	if rt == nil {
		return
	}
	e := engine.New(parent, nil)
	rt.RegisterEngineShutdown(e)
}
