package compact

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
	"github.com/2456868764/rabbit-code/internal/tools/webfetchtool"
	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
	"github.com/2456868764/rabbit-code/internal/utils/shell"
)

// PinnedCacheEdits mirrors microCompact.ts pin payload (user message index + cache_edits block JSON).
type PinnedCacheEdits struct {
	UserMessageIndex int
	Block            json.RawMessage
}

// MicrocompactEditBuffer holds pending/pinned cache_edits for the API layer (cachedMicrocompact.ts state subset).
type MicrocompactEditBuffer struct {
	mu             sync.Mutex
	pending        json.RawMessage
	pinned         []PinnedCacheEdits
	toolsSentToAPI bool
}

func (b *MicrocompactEditBuffer) ConsumePendingCacheEdits() json.RawMessage {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	edits := b.pending
	b.pending = nil
	return edits
}

func (b *MicrocompactEditBuffer) GetPinnedCacheEdits() []PinnedCacheEdits {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]PinnedCacheEdits, len(b.pinned))
	copy(out, b.pinned)
	return out
}

func (b *MicrocompactEditBuffer) PinCacheEdits(userMessageIndex int, block json.RawMessage) {
	if b == nil || len(block) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := json.RawMessage(append([]byte(nil), block...))
	b.pinned = append(b.pinned, PinnedCacheEdits{UserMessageIndex: userMessageIndex, Block: cp})
}

func (b *MicrocompactEditBuffer) SetPendingCacheEdits(block json.RawMessage) {
	if b == nil || len(block) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = json.RawMessage(append([]byte(nil), block...))
}

// MarkToolsSentToAPIState mirrors markToolsSentToAPIState (headless flag).
func (b *MicrocompactEditBuffer) MarkToolsSentToAPIState() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.toolsSentToAPI = true
}

func (b *MicrocompactEditBuffer) ToolsSentToAPI() bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.toolsSentToAPI
}

// ResetMicrocompactState mirrors resetMicrocompactState.
func (b *MicrocompactEditBuffer) ResetMicrocompactState() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = nil
	b.pinned = nil
	b.toolsSentToAPI = false
}

// compactableTools mirrors microCompact.ts COMPACTABLE_TOOLS (assembled from src/tools/* constants).
var compactableTools = buildCompactableToolSet()

func buildCompactableToolSet() map[string]struct{} {
	m := make(map[string]struct{})
	add := func(s string) { m[s] = struct{}{} }
	add(filereadtool.FileReadToolName)
	for _, n := range shell.ShellToolNames() {
		add(n)
	}
	add(greptool.GrepToolName)
	add(globtool.GlobToolName)
	add(websearchtool.WebSearchToolName)
	add(webfetchtool.WebFetchToolName)
	add(fileedittool.FileEditToolName)
	add(filewritetool.FileWriteToolName)
	return m
}

// IsCompactableToolName reports whether name is in the upstream COMPACTABLE_TOOLS set.
func IsCompactableToolName(name string) bool {
	_, ok := compactableTools[name]
	return ok
}

// CollectCompactableToolUseIDsFromTranscriptJSON returns tool_use ids for compactable tools in assistant
// messages (microCompact.ts collectCompactableToolIds).
func CollectCompactableToolUseIDsFromTranscriptJSON(transcript []byte) ([]string, error) {
	var arr []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range arr {
		if m.Role != "assistant" || len(m.Content) == 0 {
			continue
		}
		var blocks []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(m.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_use" && b.ID != "" && IsCompactableToolName(b.Name) {
				ids = append(ids, b.ID)
			}
		}
	}
	return ids, nil
}

// IsMainThreadQuerySource mirrors microCompact.ts isMainThreadSource (repl_main_thread prefix for output styles).
func IsMainThreadQuerySource(querySource string) bool {
	s := strings.TrimSpace(querySource)
	return s == "" || strings.HasPrefix(s, "repl_main_thread")
}
