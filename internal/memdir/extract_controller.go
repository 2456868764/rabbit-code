package memdir

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

// ExtractController coordinates background memory extraction (initExtractMemories closure in extractMemories.ts).
type ExtractController struct {
	mu sync.Mutex
	wg sync.WaitGroup

	lastMessageUUID string
	inProgress      bool
	pending         *extractPending
	turnsSince      int
}

type extractPending struct {
	MessagesJSON json.RawMessage
	Merged       map[string]interface{}
}

// ExtractHookArgs is passed from the engine stop hook into the controller.
type ExtractHookArgs struct {
	LoopErr        error
	MessagesJSON   json.RawMessage
	MemoryDir      string
	Merged         map[string]interface{}
	NonInteractive bool
	AgentID        string
	Deps           querydeps.Deps
	Model          string
	MaxTokens      int
	OnMemorySaved  func(memoryPaths []string, teamCount int)
	UUIDField      string // default query.RabbitMessageUUIDKey
	IsTrailingRun  bool
}

// HandleStopHook mirrors executeExtractMemories (fire-and-forget from the engine).
func (c *ExtractController) HandleStopHook(ctx context.Context, a ExtractHookArgs) {
	if c == nil {
		return
	}
	if a.LoopErr != nil {
		return
	}
	if strings.TrimSpace(a.AgentID) != "" {
		return
	}
	if !features.ExtractMemoriesAllowed(a.NonInteractive) {
		return
	}
	if features.RemoteModeWithoutMemoryDir() {
		return
	}
	memDir := strings.TrimSpace(a.MemoryDir)
	if memDir == "" {
		return
	}
	if !features.AutoMemoryEnabledFromMerged(a.Merged) {
		return
	}

	uuidField := strings.TrimSpace(a.UUIDField)
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}

	c.mu.Lock()
	if c.inProgress {
		c.pending = &extractPending{MessagesJSON: cloneRaw(a.MessagesJSON), Merged: a.Merged}
		c.mu.Unlock()
		return
	}
	if !a.IsTrailingRun {
		c.turnsSince++
		if c.turnsSince < features.ExtractMemoriesInterval() {
			c.mu.Unlock()
			return
		}
	}
	c.turnsSince = 0
	sinceUUID := c.lastMessageUUID
	c.mu.Unlock()

	if HasMemoryWritesSince(a.MessagesJSON, sinceUUID, memDir, uuidField) {
		c.mu.Lock()
		if u := LastEmbeddedMessageUUID(a.MessagesJSON, uuidField); u != "" {
			c.lastMessageUUID = u
		}
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	c.inProgress = true
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.execAndMaybeTrail(ctx, a, uuidField, memDir)
	}()
}

func cloneRaw(r json.RawMessage) json.RawMessage {
	if len(r) == 0 {
		return nil
	}
	return json.RawMessage(append([]byte(nil), r...))
}

func (c *ExtractController) execAndMaybeTrail(ctx context.Context, a ExtractHookArgs, uuidField, memDir string) {
	defer func() {
		c.mu.Lock()
		c.inProgress = false
		p := c.pending
		c.pending = nil
		c.mu.Unlock()
		if p != nil {
			next := a
			next.MessagesJSON = p.MessagesJSON
			next.Merged = p.Merged
			next.IsTrailingRun = true
			c.execAndMaybeTrail(ctx, next, uuidField, memDir)
		}
	}()

	c.mu.Lock()
	since := c.lastMessageUUID
	c.mu.Unlock()
	newCount := CountModelVisibleMessagesSince(a.MessagesJSON, since, uuidField)

	headers, err := ScanMemoryFiles(ctx, memDir)
	if err != nil {
		headers = nil
	}
	manifest := FormatMemoryManifest(headers)

	skipIdx := features.ExtractMemoriesSkipIndex()
	var userPrompt string
	if features.TeamMemoryEnabledFromMerged(a.Merged) {
		userPrompt = BuildExtractCombinedPrompt(newCount, manifest, skipIdx, a.Merged)
	} else {
		userPrompt = BuildExtractAutoOnlyPrompt(newCount, manifest, skipIdx)
	}

	dep := ForkedExtractDeps{
		Tools:     a.Deps.Tools,
		Turn:      a.Deps.Turn,
		Model:     a.Model,
		MaxTokens: a.MaxTokens,
	}
	res, err := RunForkedExtractMemory(ctx, dep, ForkedExtractParams{
		ParentMessagesJSON: a.MessagesJSON,
		UserPrompt:         userPrompt,
		MemoryDir:          memDir,
		MaxTurns:           5,
		QuerySource:        query.QuerySourceExtractMemories,
		NonInteractive:     a.NonInteractive,
		Merged:             a.Merged,
	})
	if err != nil {
		return
	}

	c.mu.Lock()
	if u := LastEmbeddedMessageUUID(a.MessagesJSON, uuidField); u != "" {
		c.lastMessageUUID = u
	}
	c.mu.Unlock()

	if len(res.MemoryFilePaths) > 0 && a.OnMemorySaved != nil {
		a.OnMemorySaved(res.MemoryFilePaths, res.TeamMemoryWrites)
	}
}

// Wait blocks until in-flight extractions finish or ctx is done.
func (c *ExtractController) Wait(ctx context.Context) {
	if c == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
	case <-done:
	}
}
