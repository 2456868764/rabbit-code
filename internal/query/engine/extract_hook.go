package engine

import (
	"context"

	"github.com/2456868764/rabbit-code/internal/memdir"
	"github.com/2456868764/rabbit-code/internal/query"
)

func (e *Engine) stopHookExtractMemories(ctx context.Context, st query.LoopState, loopErr error) {
	if e.extractCtl == nil || !e.useQueryLoop() {
		return
	}
	memDir := e.memdirMemoryDir
	if memDir == "" {
		return
	}
	args := memdir.ExtractHookArgs{
		LoopErr:        loopErr,
		MessagesJSON:   st.MessagesJSON,
		MemoryDir:      memDir,
		Merged:         e.initialSettings,
		NonInteractive: e.nonInteractive,
		AgentID:        e.agentID,
		Deps:           e.deps,
		Model:          e.model,
		MaxTokens:      e.maxTokens,
	}
	if e.extractMemoriesSavedFn != nil {
		fn := e.extractMemoriesSavedFn
		args.OnMemorySaved = func(paths []string, team int) { fn(paths, team) }
	}
	e.extractCtl.HandleStopHook(ctx, args)
}

// DrainExtractMemories waits for in-flight forked extract loops (print.ts drainPendingExtraction analogue).
func (e *Engine) DrainExtractMemories(ctx context.Context) {
	if e == nil || e.extractCtl == nil {
		return
	}
	e.extractCtl.Wait(ctx)
}
