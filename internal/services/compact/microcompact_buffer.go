package compact

import (
	"encoding/json"
	"sync"
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
