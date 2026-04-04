package compact

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/2456868764/rabbit-code/internal/features"
)

// SessionMemoryCompactConfig mirrors sessionMemoryCompact.ts SessionMemoryCompactConfig.
type SessionMemoryCompactConfig struct {
	MinTokens            int
	MinTextBlockMessages int
	MaxTokens            int
}

// DefaultSessionMemoryCompactConfig mirrors DEFAULT_SM_COMPACT_CONFIG.
var DefaultSessionMemoryCompactConfig = SessionMemoryCompactConfig{
	MinTokens:            10_000,
	MinTextBlockMessages: 5,
	MaxTokens:            40_000,
}

// DEFAULT_SM_COMPACT_CONFIG mirrors the sessionMemoryCompact.ts export name (same struct as DefaultSessionMemoryCompactConfig).
var DEFAULT_SM_COMPACT_CONFIG = DefaultSessionMemoryCompactConfig

var smCfgMu sync.RWMutex
var smCfgOverride *SessionMemoryCompactConfig

// GetSessionMemoryCompactConfig returns override if set, else env-backed defaults (GrowthBook analogue).
func GetSessionMemoryCompactConfig() SessionMemoryCompactConfig {
	smCfgMu.RLock()
	o := smCfgOverride
	smCfgMu.RUnlock()
	if o != nil {
		return *o
	}
	return SessionMemoryCompactConfig{
		MinTokens:            features.SessionMemoryCompactMinTokens(),
		MinTextBlockMessages: features.SessionMemoryCompactMinTextBlockMessages(),
		MaxTokens:            features.SessionMemoryCompactMaxTokens(),
	}
}

// SetSessionMemoryCompactConfig merges partial config (sessionMemoryCompact.ts setSessionMemoryCompactConfig).
func SetSessionMemoryCompactConfig(partial SessionMemoryCompactConfig) {
	smCfgMu.Lock()
	defer smCfgMu.Unlock()
	var base SessionMemoryCompactConfig
	if smCfgOverride != nil {
		base = *smCfgOverride
	} else {
		base = SessionMemoryCompactConfig{
			MinTokens:            features.SessionMemoryCompactMinTokens(),
			MinTextBlockMessages: features.SessionMemoryCompactMinTextBlockMessages(),
			MaxTokens:            features.SessionMemoryCompactMaxTokens(),
		}
	}
	if partial.MinTokens > 0 {
		base.MinTokens = partial.MinTokens
	}
	if partial.MinTextBlockMessages > 0 {
		base.MinTextBlockMessages = partial.MinTextBlockMessages
	}
	if partial.MaxTokens > 0 {
		base.MaxTokens = partial.MaxTokens
	}
	cp := base
	smCfgOverride = &cp
}

// ResetSessionMemoryCompactConfig clears override (sessionMemoryCompact.ts resetSessionMemoryCompactConfig).
func ResetSessionMemoryCompactConfig() {
	smCfgMu.Lock()
	defer smCfgMu.Unlock()
	smCfgOverride = nil
}

// SessionMemorySectionMaxTokens mirrors SessionMemory/prompts.ts MAX_SECTION_LENGTH for per-section truncation.
const SessionMemorySectionMaxTokens = 2000

// TruncateSessionMemoryForCompact mirrors truncateSessionMemoryForCompact (SessionMemory/prompts.ts).
func TruncateSessionMemoryForCompact(content string) (truncated string, wasTruncated bool) {
	lines := strings.Split(content, "\n")
	maxChars := SessionMemorySectionMaxTokens * 4
	var out []string
	var curHeader string
	var curLines []string
	flush := func() {
		if curHeader == "" {
			out = append(out, curLines...)
			curLines = nil
			return
		}
		section := strings.Join(curLines, "\n")
		if len(section) <= maxChars {
			out = append(out, curHeader)
			out = append(out, curLines...)
		} else {
			var kept []string
			n := 0
			for _, ln := range curLines {
				if n+len(ln)+1 > maxChars {
					wasTruncated = true
					break
				}
				kept = append(kept, ln)
				n += len(ln) + 1
			}
			out = append(out, curHeader)
			out = append(out, kept...)
			wasTruncated = wasTruncated || len(kept) < len(curLines)
		}
		curLines = nil
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			flush()
			curHeader = line
		} else {
			curLines = append(curLines, line)
		}
	}
	flush()
	return strings.Join(out, "\n"), wasTruncated
}

// ShouldUseSessionMemoryCompaction mirrors shouldUseSessionMemoryCompaction().
func ShouldUseSessionMemoryCompaction() bool {
	return features.SessionMemoryCompactionEnabled()
}

// SessionMemoryCompactHooks supplies host/session-memory I/O for TrySessionMemoryCompactionTranscriptJSON.
type SessionMemoryCompactHooks struct {
	WaitForSessionMemoryExtraction func(ctx context.Context) error
	GetSessionMemoryContent        func(ctx context.Context) (string, error)
	IsSessionMemoryEmpty           func(ctx context.Context, content string) (bool, error)
	GetLastSummarizedMessageUUID   func() string
	SessionStartHooks              func(ctx context.Context, model string) ([]json.RawMessage, error)
	TranscriptPath                 func() string
	SessionMemoryPathForFooter     func() string
	PlanAttachmentMessageJSON      func(agentID string) (json.RawMessage, error)
	OnSuccessfulCompaction         func()
}

// NewSessionMemoryCompactExecutor returns an engine.SessionMemoryCompact-compatible closure (bind hooks once).
func NewSessionMemoryCompactExecutor(hooks SessionMemoryCompactHooks) func(ctx context.Context, agentID, model string, autoCompactThreshold int, transcript []byte) ([]byte, bool, error) {
	return func(ctx context.Context, agentID, model string, autoCompactThreshold int, transcript []byte) ([]byte, bool, error) {
		return TrySessionMemoryCompactionTranscriptJSON(ctx, transcript, model, agentID, autoCompactThreshold, &hooks)
	}
}

// TrySessionMemoryCompactionTranscriptJSON mirrors trySessionMemoryCompaction (Messages API JSON array transcript).
func TrySessionMemoryCompactionTranscriptJSON(ctx context.Context, transcript []byte, model, agentID string, autoCompactThreshold int, hooks *SessionMemoryCompactHooks) (replacement []byte, ok bool, err error) {
	if !ShouldUseSessionMemoryCompaction() || hooks == nil || hooks.GetSessionMemoryContent == nil {
		return nil, false, nil
	}
	if hooks.WaitForSessionMemoryExtraction != nil {
		if err := hooks.WaitForSessionMemoryExtraction(ctx); err != nil {
			return nil, false, err
		}
	}
	sm, err := hooks.GetSessionMemoryContent(ctx)
	if err != nil {
		return nil, false, nil
	}
	if strings.TrimSpace(sm) == "" {
		return nil, false, nil
	}
	if hooks.IsSessionMemoryEmpty != nil {
		empty, eerr := hooks.IsSessionMemoryEmpty(ctx, sm)
		if eerr != nil {
			return nil, false, eerr
		}
		if empty {
			return nil, false, nil
		}
	}

	var lines []json.RawMessage
	if err := json.Unmarshal(transcript, &lines); err != nil {
		return nil, false, err
	}
	if len(lines) == 0 {
		return nil, false, nil
	}

	lastSummarized := ""
	if hooks.GetLastSummarizedMessageUUID != nil {
		lastSummarized = strings.TrimSpace(hooks.GetLastSummarizedMessageUUID())
	}
	var lastSummarizedIdx int
	if lastSummarized != "" {
		lastSummarizedIdx = indexOfTopLevelUUID(lines, lastSummarized)
		if lastSummarizedIdx < 0 {
			return nil, false, nil
		}
	} else {
		lastSummarizedIdx = len(lines) - 1
	}

	start, err := CalculateSessionMemoryKeepStartIndex(lines, lastSummarizedIdx)
	if err != nil {
		return nil, false, err
	}
	var keep []json.RawMessage
	for i := start; i < len(lines); i++ {
		var m map[string]interface{}
		if json.Unmarshal(lines[i], &m) != nil {
			continue
		}
		if IsCompactBoundaryMessageMap(m) {
			continue
		}
		keep = append(keep, lines[i])
	}

	preTok, _ := EstimateMessageTokensFromAPIMessagesJSON(transcript)
	lastParent := LastTopLevelMessageUUIDTranscriptJSON(transcript)
	boundary, err := CreateCompactBoundaryMessageJSON("auto", preTok, lastParent, "", 0)
	if err != nil {
		return nil, false, err
	}
	names, err := ExtractDiscoveredToolNamesFromTranscriptJSON(transcript)
	if err != nil {
		return nil, false, err
	}
	boundary, err = AttachPreCompactDiscoveredToolsToBoundaryJSON(boundary, names)
	if err != nil {
		return nil, false, err
	}

	truncSM, trunc := TruncateSessionMemoryForCompact(sm)
	tp := ""
	if hooks.TranscriptPath != nil {
		tp = hooks.TranscriptPath()
	}
	sumText := GetCompactUserSummaryMessage(truncSM, true, tp, true)
	if trunc {
		mp := ""
		if hooks.SessionMemoryPathForFooter != nil {
			mp = hooks.SessionMemoryPathForFooter()
		}
		if mp != "" {
			sumText += "\n\nSome session memory sections were truncated for length. The full session memory can be viewed at: " + mp
		}
	}
	summaryUUID := smRandomUUIDv4()
	sumMsg, err := smUserSummaryMessageJSON(sumText, summaryUUID)
	if err != nil {
		return nil, false, err
	}

	var head, tail string
	if len(keep) > 0 {
		head = topLevelUUIDFromRaw(keep[0])
		tail = topLevelUUIDFromRaw(keep[len(keep)-1])
	}
	if head != "" && tail != "" {
		boundary, err = AnnotateBoundaryWithPreservedSegmentJSON(boundary, summaryUUID, head, tail)
		if err != nil {
			return nil, false, err
		}
	}

	var hookResults []json.RawMessage
	if hooks.SessionStartHooks != nil {
		hookResults, err = hooks.SessionStartHooks(ctx, model)
		if err != nil {
			return nil, false, err
		}
	}

	var attachments []json.RawMessage
	if hooks.PlanAttachmentMessageJSON != nil {
		planRaw, perr := hooks.PlanAttachmentMessageJSON(agentID)
		if perr != nil {
			return nil, false, perr
		}
		if len(bytesTrimSpaceJSON(planRaw)) > 0 {
			attachments = append(attachments, planRaw)
		}
	}

	out, err := BuildPostCompactMessagesJSON(boundary, []json.RawMessage{sumMsg}, keep, attachments, hookResults)
	if err != nil {
		return nil, false, err
	}

	if autoCompactThreshold > 0 {
		postTok, estErr := EstimateMessageTokensFromAPIMessagesJSON(out)
		if estErr == nil && postTok >= autoCompactThreshold {
			return nil, false, nil
		}
	}

	if hooks.OnSuccessfulCompaction != nil {
		hooks.OnSuccessfulCompaction()
	}
	return out, true, nil
}

func bytesTrimSpaceJSON(m json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(m)))
}

func smRandomUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("sm-%d", len(b))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func smUserSummaryMessageJSON(text, uuid string) (json.RawMessage, error) {
	m := map[string]interface{}{
		"role":    "user",
		"uuid":    uuid,
		"content": []interface{}{map[string]string{"type": "text", "text": text}},
	}
	return json.Marshal(m)
}

func topLevelUUIDFromRaw(raw json.RawMessage) string {
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	s, _ := m["uuid"].(string)
	return strings.TrimSpace(s)
}

func indexOfTopLevelUUID(lines []json.RawMessage, uuid string) int {
	u := strings.TrimSpace(uuid)
	if u == "" {
		return -1
	}
	for i, ln := range lines {
		if topLevelUUIDFromRaw(ln) == u {
			return i
		}
	}
	return -1
}

// HasTextBlocks mirrors sessionMemoryCompact.ts hasTextBlocks for one top-level Messages-API message JSON object.
func HasTextBlocks(messageJSON json.RawMessage) bool {
	var m map[string]interface{}
	if json.Unmarshal(messageJSON, &m) != nil {
		return false
	}
	return HasTextBlocksTranscriptLine(m)
}

// HasTextBlocksTranscriptLine mirrors hasTextBlocks for one decoded transcript element (user/assistant).
func HasTextBlocksTranscriptLine(m map[string]interface{}) bool {
	if m == nil {
		return false
	}
	role, _ := m["role"].(string)
	typ, _ := m["type"].(string)
	isAsst := role == "assistant" || typ == "assistant"
	isUser := role == "user" || typ == "user"
	var content interface{}
	if msg, ok := m["message"].(map[string]interface{}); ok {
		content = msg["content"]
	} else {
		content = m["content"]
	}
	if isAsst {
		arr, ok := content.([]interface{})
		if !ok {
			return false
		}
		for _, b := range arr {
			bm, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			if t, _ := bm["type"].(string); t == "text" {
				if s, _ := bm["text"].(string); strings.TrimSpace(s) != "" {
					return true
				}
			}
		}
		return false
	}
	if isUser {
		switch c := content.(type) {
		case string:
			return strings.TrimSpace(c) != ""
		case []interface{}:
			for _, b := range c {
				bm, ok := b.(map[string]interface{})
				if !ok {
					continue
				}
				if t, _ := bm["type"].(string); t == "text" {
					if s, _ := bm["text"].(string); strings.TrimSpace(s) != "" {
						return true
					}
				}
			}
		}
	}
	return false
}

func smMessageMap(raw json.RawMessage) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func smIsUserLine(m map[string]interface{}) bool {
	if r, ok := m["role"].(string); ok && r == "user" {
		return true
	}
	if t, ok := m["type"].(string); ok && t == "user" {
		return true
	}
	return false
}

func smIsAssistantLine(m map[string]interface{}) bool {
	if r, ok := m["role"].(string); ok && r == "assistant" {
		return true
	}
	if t, ok := m["type"].(string); ok && t == "assistant" {
		return true
	}
	return false
}

func getToolResultIDsFromUserMap(m map[string]interface{}) []string {
	var content interface{}
	if msg, ok := m["message"].(map[string]interface{}); ok {
		content = msg["content"]
	} else {
		content = m["content"]
	}
	arr, ok := content.([]interface{})
	if !ok {
		return nil
	}
	var ids []string
	for _, b := range arr {
		bm, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := bm["type"].(string); t != "tool_result" {
			continue
		}
		if id, _ := bm["tool_use_id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func hasToolUseWithIDsAssistantMap(m map[string]interface{}, needed map[string]struct{}) bool {
	if !smIsAssistantLine(m) || len(needed) == 0 {
		return false
	}
	var content interface{}
	if msg, ok := m["message"].(map[string]interface{}); ok {
		content = msg["content"]
	} else {
		content = m["content"]
	}
	arr, ok := content.([]interface{})
	if !ok {
		return false
	}
	for _, b := range arr {
		bm, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := bm["type"].(string); t != "tool_use" {
			continue
		}
		id, _ := bm["id"].(string)
		if _, ok := needed[id]; ok {
			return true
		}
	}
	return false
}

func assistantMessageIDsInLine(m map[string]interface{}) string {
	if !smIsAssistantLine(m) {
		return ""
	}
	if msg, ok := m["message"].(map[string]interface{}); ok {
		if id, ok := msg["id"].(string); ok {
			return id
		}
	}
	if id, ok := m["id"].(string); ok {
		return id
	}
	return ""
}

// AdjustIndexToPreserveAPIInvariants mirrors sessionMemoryCompact.ts adjustIndexToPreserveAPIInvariants (transcript as []json.RawMessage lines).
func AdjustIndexToPreserveAPIInvariants(lines []json.RawMessage, startIndex int) (int, error) {
	return AdjustStartIndexToPreserveAPIInvariants(lines, startIndex)
}

// AdjustStartIndexToPreserveAPIInvariants mirrors adjustIndexToPreserveAPIInvariants (tool pairs + shared assistant id).
func AdjustStartIndexToPreserveAPIInvariants(lines []json.RawMessage, startIndex int) (int, error) {
	if startIndex <= 0 || startIndex >= len(lines) {
		return startIndex, nil
	}
	adjusted := startIndex

	var allToolResults []string
	for i := adjusted; i < len(lines); i++ {
		m, err := smMessageMap(lines[i])
		if err != nil {
			continue
		}
		if smIsUserLine(m) {
			allToolResults = append(allToolResults, getToolResultIDsFromUserMap(m)...)
		}
	}
	if len(allToolResults) > 0 {
		inKept := make(map[string]struct{})
		for i := adjusted; i < len(lines); i++ {
			m, err := smMessageMap(lines[i])
			if err != nil || !smIsAssistantLine(m) {
				continue
			}
			var content interface{}
			if msg, ok := m["message"].(map[string]interface{}); ok {
				content = msg["content"]
			} else {
				content = m["content"]
			}
			arr, ok := content.([]interface{})
			if !ok {
				continue
			}
			for _, b := range arr {
				bm, ok := b.(map[string]interface{})
				if !ok {
					continue
				}
				if t, _ := bm["type"].(string); t == "tool_use" {
					if id, _ := bm["id"].(string); id != "" {
						inKept[id] = struct{}{}
					}
				}
			}
		}
		needed := make(map[string]struct{})
		for _, id := range allToolResults {
			if _, ok := inKept[id]; !ok {
				needed[id] = struct{}{}
			}
		}
		for i := adjusted - 1; i >= 0 && len(needed) > 0; i-- {
			m, err := smMessageMap(lines[i])
			if err != nil {
				continue
			}
			if hasToolUseWithIDsAssistantMap(m, needed) {
				adjusted = i
				var content interface{}
				if msg, ok := m["message"].(map[string]interface{}); ok {
					content = msg["content"]
				} else {
					content = m["content"]
				}
				if arr, ok := content.([]interface{}); ok {
					for _, b := range arr {
						bm, ok := b.(map[string]interface{})
						if !ok {
							continue
						}
						if t, _ := bm["type"].(string); t == "tool_use" {
							if id, _ := bm["id"].(string); id != "" {
								delete(needed, id)
							}
						}
					}
				}
			}
		}
	}

	msgIDs := make(map[string]struct{})
	for i := adjusted; i < len(lines); i++ {
		m, err := smMessageMap(lines[i])
		if err != nil || !smIsAssistantLine(m) {
			continue
		}
		if mid := assistantMessageIDsInLine(m); mid != "" {
			msgIDs[mid] = struct{}{}
		}
	}
	for i := adjusted - 1; i >= 0; i-- {
		m, err := smMessageMap(lines[i])
		if err != nil || !smIsAssistantLine(m) {
			continue
		}
		mid := assistantMessageIDsInLine(m)
		if mid != "" {
			if _, ok := msgIDs[mid]; ok {
				adjusted = i
			}
		}
	}
	return adjusted, nil
}

// CalculateSessionMemoryKeepStartIndex mirrors calculateMessagesToKeepIndex.
func CalculateSessionMemoryKeepStartIndex(lines []json.RawMessage, lastSummarizedIndex int) (int, error) {
	if len(lines) == 0 {
		return 0, nil
	}
	cfg := GetSessionMemoryCompactConfig()
	start := len(lines)
	if lastSummarizedIndex >= 0 {
		start = lastSummarizedIndex + 1
	}
	if start > len(lines) {
		start = len(lines)
	}

	estimateOne := func(raw json.RawMessage) (int, error) {
		wrap, err := json.Marshal([]json.RawMessage{raw})
		if err != nil {
			return 0, err
		}
		return EstimateMessageTokensFromAPIMessagesJSON(wrap)
	}

	totalTok := 0
	textCount := 0
	for i := start; i < len(lines); i++ {
		m, err := smMessageMap(lines[i])
		if err != nil {
			continue
		}
		tok, err := estimateOne(lines[i])
		if err != nil {
			return 0, err
		}
		totalTok += tok
		if HasTextBlocksTranscriptLine(m) {
			textCount++
		}
	}
	if totalTok >= cfg.MaxTokens {
		return AdjustStartIndexToPreserveAPIInvariants(lines, start)
	}
	if totalTok >= cfg.MinTokens && textCount >= cfg.MinTextBlockMessages {
		return AdjustStartIndexToPreserveAPIInvariants(lines, start)
	}

	floor := 0
	if idx, err := FindLastCompactBoundaryIndexTranscriptJSON(jsonLinesArray(lines)); err == nil && idx >= 0 {
		floor = idx + 1
	}
	for i := start - 1; i >= floor; i-- {
		m, err := smMessageMap(lines[i])
		if err != nil {
			continue
		}
		tok, err := estimateOne(lines[i])
		if err != nil {
			return 0, err
		}
		totalTok += tok
		if HasTextBlocksTranscriptLine(m) {
			textCount++
		}
		start = i
		if totalTok >= cfg.MaxTokens {
			break
		}
		if totalTok >= cfg.MinTokens && textCount >= cfg.MinTextBlockMessages {
			break
		}
	}
	return AdjustStartIndexToPreserveAPIInvariants(lines, start)
}

// CalculateMessagesToKeepIndex mirrors sessionMemoryCompact.ts calculateMessagesToKeepIndex.
func CalculateMessagesToKeepIndex(lines []json.RawMessage, lastSummarizedIndex int) (int, error) {
	return CalculateSessionMemoryKeepStartIndex(lines, lastSummarizedIndex)
}

func jsonLinesArray(lines []json.RawMessage) []byte {
	b, err := json.Marshal(lines)
	if err != nil {
		return []byte("[]")
	}
	return b
}
