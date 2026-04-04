package compact

import (
	"encoding/json"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// cachedMicrocompactState mirrors cachedMicrocompact.ts CachedMCState (tool order + deletions).
type cachedMicrocompactState struct {
	order   []string
	seen    map[string]struct{}
	deleted map[string]struct{}
}

func newCachedMicrocompactState() *cachedMicrocompactState {
	return &cachedMicrocompactState{
		seen:    make(map[string]struct{}),
		deleted: make(map[string]struct{}),
	}
}

func (s *cachedMicrocompactState) reset() {
	if s == nil {
		return
	}
	s.order = nil
	s.seen = make(map[string]struct{})
	s.deleted = make(map[string]struct{})
}

func (s *cachedMicrocompactState) registerToolUseID(id string) {
	if s == nil || id == "" {
		return
	}
	if _, ok := s.seen[id]; ok {
		return
	}
	s.seen[id] = struct{}{}
	s.order = append(s.order, id)
}

func (s *cachedMicrocompactState) markDeleted(ids []string) {
	if s == nil {
		return
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		s.deleted[id] = struct{}{}
	}
}

func (s *cachedMicrocompactState) activeIDs() []string {
	if s == nil {
		return nil
	}
	var out []string
	for _, id := range s.order {
		if _, d := s.deleted[id]; d {
			continue
		}
		out = append(out, id)
	}
	return out
}

func (s *cachedMicrocompactState) toolIDsToDelete(trigger, keepRecent int) []string {
	active := s.activeIDs()
	if len(active) <= trigger {
		return nil
	}
	nRemove := len(active) - keepRecent
	if nRemove <= 0 {
		return nil
	}
	if nRemove > len(active) {
		nRemove = len(active)
	}
	return append([]string(nil), active[:nRemove]...)
}

func ingestToolResultsIntoCachedState(s *cachedMicrocompactState, transcript []byte) error {
	if s == nil {
		return nil
	}
	idToName, err := toolUseIDToCompactableName(transcript)
	if err != nil {
		return err
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return err
	}
	for _, m := range arr {
		if !smIsUserLine(m) {
			continue
		}
		var content interface{}
		if msg, ok := m["message"].(map[string]interface{}); ok {
			content = msg["content"]
		} else {
			content = m["content"]
		}
		blocks, ok := content.([]interface{})
		if !ok {
			continue
		}
		for _, b := range blocks {
			bm, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			if typ, _ := bm["type"].(string); typ != "tool_result" {
				continue
			}
			tid, _ := bm["tool_use_id"].(string)
			if tid == "" {
				continue
			}
			name, ok := idToName[tid]
			if !ok || !IsCompactableToolName(name) {
				continue
			}
			s.registerToolUseID(tid)
		}
	}
	return nil
}

func toolUseIDToCompactableName(transcript []byte) (map[string]string, error) {
	out := make(map[string]string)
	var arr []map[string]interface{}
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return nil, err
	}
	for _, m := range arr {
		if !smIsAssistantLine(m) {
			continue
		}
		var content interface{}
		if msg, ok := m["message"].(map[string]interface{}); ok {
			content = msg["content"]
		} else {
			content = m["content"]
		}
		blocks, ok := content.([]interface{})
		if !ok {
			continue
		}
		for _, b := range blocks {
			bm, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			if typ, _ := bm["type"].(string); typ != "tool_use" {
				continue
			}
			id, _ := bm["id"].(string)
			name, _ := bm["name"].(string)
			if id != "" && name != "" && IsCompactableToolName(name) {
				out[id] = name
			}
		}
	}
	return out, nil
}

// CachedMicrocompactModelSupported mirrors isModelSupportedForCacheEditing (headless: Claude models).
func CachedMicrocompactModelSupported(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	return strings.Contains(m, "claude")
}

// RunCachedMicrocompactTranscriptJSON mirrors cachedMicrocompactPath side effects (pending cache_edits on buffer).
func RunCachedMicrocompactTranscriptJSON(transcript []byte, querySource, model string, buf *MicrocompactEditBuffer) (*MicrocompactCompactionInfo, error) {
	if buf == nil || len(transcript) == 0 {
		return nil, nil
	}
	if !features.CachedMicrocompactEnabled() || !IsMainThreadQuerySource(querySource) || !CachedMicrocompactModelSupported(model) {
		return nil, nil
	}
	st := buf.ensureCachedState()
	if err := ingestToolResultsIntoCachedState(st, transcript); err != nil {
		return nil, err
	}
	del := st.toolIDsToDelete(features.CachedMicrocompactTriggerThreshold(), features.CachedMicrocompactKeepRecent())
	if len(del) == 0 {
		return nil, nil
	}
	st.markDeleted(del)
	pe := &MicrocompactPendingCacheEdits{
		Trigger:        "cached_microcompact",
		DeletedToolIDs: append([]string(nil), del...),
	}
	raw, err := json.Marshal(pe)
	if err != nil {
		return nil, err
	}
	buf.SetPendingCacheEdits(raw)
	SuppressCompactWarning()
	return &MicrocompactCompactionInfo{PendingCacheEdits: pe}, nil
}
