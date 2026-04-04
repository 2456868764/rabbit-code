package compact

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

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
	cached         *cachedMicrocompactState
}

func (b *MicrocompactEditBuffer) ensureCachedState() *cachedMicrocompactState {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cached == nil {
		b.cached = newCachedMicrocompactState()
	}
	return b.cached
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
	if b.cached != nil {
		b.cached.reset()
	}
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
	if b.cached != nil {
		b.cached.reset()
	}
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

// MicrocompactPendingCacheEdits mirrors microCompact.ts PendingCacheEdits (for compactionInfo when cached MC is wired).
type MicrocompactPendingCacheEdits struct {
	Trigger                    string   `json:"trigger"`
	DeletedToolIDs             []string `json:"deletedToolIds,omitempty"`
	BaselineCacheDeletedTokens int      `json:"baselineCacheDeletedTokens,omitempty"`
}

// MicrocompactCompactionInfo mirrors microCompact.ts compactionInfo subset.
type MicrocompactCompactionInfo struct {
	PendingCacheEdits *MicrocompactPendingCacheEdits `json:"pendingCacheEdits,omitempty"`
}

// MicrocompactMessagesAPIJSON mirrors microCompact.ts microcompactMessages for Anthropic Messages API transcript JSON:
// clears compact-warning suppression, runs time-based microcompact first (short-circuit with side effects), then returns
// messages unchanged for the cached-microcompact path until cachedMicrocompact.ts state is ported (see doc).
func MicrocompactMessagesAPIJSON(messagesJSON []byte, querySource string, now, lastAssistantAt time.Time, mainLoopModel string, buf *MicrocompactEditBuffer) (out []byte, tokensSaved int, timeBasedChanged bool, compaction *MicrocompactCompactionInfo, err error) {
	ClearCompactWarningSuppression()
	out, tokensSaved, timeBasedChanged, err = RunMaybeTimeBasedMicrocompactAPIJSON(messagesJSON, querySource, now, lastAssistantAt, buf)
	if err != nil {
		return nil, 0, false, nil, err
	}
	if timeBasedChanged {
		return out, tokensSaved, true, nil, nil
	}
	if buf != nil {
		info, err := RunCachedMicrocompactTranscriptJSON(out, querySource, mainLoopModel, buf)
		if err != nil {
			return nil, 0, false, nil, err
		}
		if info != nil {
			return out, 0, false, info, nil
		}
	}
	return out, 0, false, nil, nil
}

// ImageDocumentTokenEstimate mirrors microCompact.ts IMAGE_MAX_TOKEN_SIZE (2000).
const ImageDocumentTokenEstimate = 2000

// EstimateMessageTokensFromAPIMessagesJSON mirrors microCompact.ts estimateMessageTokens for Messages API JSON
// ([{role, content}, ...]); pads by ceil(4/3) like TS. Single source for query.EstimateMessageTokensFromTranscriptJSON.
func EstimateMessageTokensFromAPIMessagesJSON(transcript []byte) (int, error) {
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return 0, err
	}
	total := 0
	for _, m := range arr {
		role := mcJSONStringField(m["role"])
		if role != "user" && role != "assistant" {
			continue
		}
		c := m["content"]
		if len(c) == 0 {
			continue
		}
		switch c[0] {
		case '"':
			var s string
			if err := json.Unmarshal(c, &s); err == nil {
				total += mcBytesAsTokens(s)
			}
		case '[':
			var blocks []map[string]json.RawMessage
			if err := json.Unmarshal(c, &blocks); err != nil {
				continue
			}
			for _, b := range blocks {
				typ := mcJSONStringField(b["type"])
				switch typ {
				case "text":
					total += mcBytesAsTokens(mcJSONStringField(b["text"]))
				case "tool_result":
					total += mcEstimateToolResultContentTokens(b["content"])
				case "image", "document":
					total += mcEstimateImageOrDocumentBlockTokens(b)
				case "thinking":
					total += mcBytesAsTokens(mcJSONStringField(b["thinking"]))
				case "redacted_thinking":
					total += mcBytesAsTokens(mcJSONStringField(b["data"]))
				case "tool_use":
					name := mcJSONStringField(b["name"])
					in := ""
					if raw, ok := b["input"]; ok && len(raw) > 0 {
						in = string(raw)
					}
					total += mcBytesAsTokens(name + in)
				default:
					total += mcBytesAsTokens(mcJSONBlockStringify(b))
				}
			}
		}
	}
	if total == 0 {
		return 0, nil
	}
	return (total*4 + 2) / 3, nil
}

func mcBytesAsTokens(s string) int {
	if s == "" {
		return 0
	}
	tok := (len(s) + 3) / 4
	if tok < 1 {
		return 1
	}
	return tok
}

// RoughTokenCountEstimationBytes mirrors roughTokenCountEstimation for a plain string (≈4 chars/token);
// compact.ts truncateToTokens / POST_COMPACT skill budgets use the same heuristic.
func RoughTokenCountEstimationBytes(s string) int {
	return mcBytesAsTokens(s)
}

func mcJSONStringField(raw json.RawMessage) string {
	if len(raw) == 0 || raw[0] != '"' {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

func mcEstimateBase64DecodedTokens(b64 string) int {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return 0
	}
	decApprox := (len(b64) * 3) / 4
	if decApprox < 0 {
		return 0
	}
	return (decApprox + 3) / 4
}

func mcEstimateImageOrDocumentBlockTokens(b map[string]json.RawMessage) int {
	n := ImageDocumentTokenEstimate
	srcRaw, ok := b["source"]
	if !ok || len(srcRaw) == 0 {
		return n
	}
	var src map[string]json.RawMessage
	if json.Unmarshal(srcRaw, &src) != nil {
		return n
	}
	data := mcJSONStringField(src["data"])
	if t := mcEstimateBase64DecodedTokens(data); t > n {
		n = t
	}
	return n
}

func mcEstimateToolResultContentTokens(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return mcBytesAsTokens(s)
		}
		return 0
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return mcBytesAsTokens(string(raw))
	}
	sum := 0
	for _, b := range arr {
		typ := mcJSONStringField(b["type"])
		switch typ {
		case "text":
			sum += mcBytesAsTokens(mcJSONStringField(b["text"]))
		case "image", "document":
			sum += mcEstimateImageOrDocumentBlockTokens(b)
		default:
			sum += mcBytesAsTokens(mcJSONBlockStringify(b))
		}
	}
	return sum
}

func mcJSONBlockStringify(b map[string]json.RawMessage) string {
	out, err := json.Marshal(b)
	if err != nil {
		return ""
	}
	return string(out)
}

// ConsumePendingCacheEdits mirrors microCompact.ts consumePendingCacheEdits (nil buffer → nil).
func ConsumePendingCacheEdits(buf *MicrocompactEditBuffer) json.RawMessage {
	if buf == nil {
		return nil
	}
	return buf.ConsumePendingCacheEdits()
}

// GetPinnedCacheEdits mirrors microCompact.ts getPinnedCacheEdits.
func GetPinnedCacheEdits(buf *MicrocompactEditBuffer) []PinnedCacheEdits {
	if buf == nil {
		return nil
	}
	return buf.GetPinnedCacheEdits()
}

// PinCacheEdits mirrors microCompact.ts pinCacheEdits.
func PinCacheEdits(buf *MicrocompactEditBuffer, userMessageIndex int, block json.RawMessage) {
	if buf == nil {
		return
	}
	buf.PinCacheEdits(userMessageIndex, block)
}

// MarkToolsSentToAPIState mirrors microCompact.ts markToolsSentToAPIState.
func MarkToolsSentToAPIState(buf *MicrocompactEditBuffer) {
	if buf == nil {
		return
	}
	buf.MarkToolsSentToAPIState()
}

// ResetMicrocompactState mirrors microCompact.ts resetMicrocompactState.
func ResetMicrocompactState(buf *MicrocompactEditBuffer) {
	if buf == nil {
		return
	}
	buf.ResetMicrocompactState()
}
