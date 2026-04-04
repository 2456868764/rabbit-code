package compact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

// RunPhase tracks compact scheduling state (services/compact/compact.ts scheduling subset, Phase 5).
type RunPhase int

const (
	RunIdle RunPhase = iota
	RunAutoPending
	RunReactivePending
	RunExecuting
)

func (p RunPhase) String() string {
	switch p {
	case RunIdle:
		return "idle"
	case RunAutoPending:
		return "auto_pending"
	case RunReactivePending:
		return "reactive_pending"
	case RunExecuting:
		return "executing"
	default:
		return "unknown"
	}
}

// ParsePhase maps engine / event strings back to RunPhase (best-effort).
func ParsePhase(s string) RunPhase {
	switch s {
	case "idle", "":
		return RunIdle
	case "auto_pending":
		return RunAutoPending
	case "reactive_pending":
		return RunReactivePending
	case "executing":
		return RunExecuting
	default:
		return RunIdle
	}
}

// Next returns the following phase for a successful scheduling edge (stub state machine).
func (p RunPhase) Next(auto, reactive bool) RunPhase {
	switch p {
	case RunIdle:
		if reactive {
			return RunReactivePending
		}
		if auto {
			return RunAutoPending
		}
		return RunIdle
	case RunAutoPending, RunReactivePending:
		return RunExecuting
	case RunExecuting:
		return RunIdle
	default:
		return RunIdle
	}
}

// AfterSuccessfulCompactExecution returns the phase after a successful executor run (H3).
func AfterSuccessfulCompactExecution(p RunPhase) RunPhase {
	if p == RunExecuting {
		return RunIdle
	}
	return p
}

// ExecutorPhaseAfterSchedule is the phase passed to CompactExecutor (pending → executing; H3).
func ExecutorPhaseAfterSchedule(scheduled RunPhase) RunPhase {
	switch scheduled {
	case RunAutoPending, RunReactivePending:
		return RunExecuting
	default:
		return scheduled
	}
}

// ResultPhaseAfterCompactExecutor is the phase for EventKindCompactResult: idle on success, else exec phase.
func ResultPhaseAfterCompactExecutor(execPhase RunPhase, execErr error) RunPhase {
	if execErr != nil {
		return execPhase
	}
	return AfterSuccessfulCompactExecution(execPhase)
}

// ExecuteStub is a Phase 5 executor that ignores transcript and returns a fixed summary (tests / wiring closure).
func ExecuteStub(_ context.Context, phase RunPhase, transcriptJSON []byte) (summary string, nextTranscriptJSON []byte, err error) {
	_ = phase
	_ = transcriptJSON
	return "[stub compact summary]", nil, nil
}

// FormatStubCompactSummary builds a deterministic summary string including transcript heuristics (tests / logging).
func FormatStubCompactSummary(phase RunPhase, transcript []byte) string {
	return fmt.Sprintf("[stub compact phase=%s bytes=%d estTok=%d]", phase.String(), len(transcript), estimateTranscriptJSONTokens(transcript))
}

// ExecuteStubWithMeta is like ExecuteStub but embeds phase and transcript metrics in the summary.
func ExecuteStubWithMeta(_ context.Context, phase RunPhase, transcriptJSON []byte) (summary string, nextTranscriptJSON []byte, err error) {
	return FormatStubCompactSummary(phase, transcriptJSON), nil, nil
}

// --- services/compact/compact.ts (constants + JSON helpers; see compact_conversation.go for boundary/partial/stream assembly) ---

// Post-compact attachment budgets (compact.ts POST_COMPACT_*).
const (
	PostCompactMaxFilesToRestore = 5
	PostCompactTokenBudget       = 50_000
	PostCompactMaxTokensPerFile  = 5_000
	PostCompactMaxTokensPerSkill = 5_000
	PostCompactSkillsTokenBudget = 25_000
	// MaxCompactStreamingRetries mirrors compact.ts MAX_COMPACT_STREAMING_RETRIES.
	MaxCompactStreamingRetries = 2
	// MaxPTLRetries mirrors compact.ts MAX_PTL_RETRIES (truncateHeadForPTLRetry).
	MaxPTLRetries = 3
	// CompactSummaryMaxOutputTokens mirrors context.ts COMPACT_MAX_OUTPUT_TOKENS (streamCompactSummary cap).
	CompactSummaryMaxOutputTokens = 20_000
)

// CompactSummarySystemPromptEnglish mirrors streamCompactSummary asSystemPrompt (compact.ts).
const CompactSummarySystemPromptEnglish = "You are a helpful AI assistant tasked with summarizing conversations."

// ErrorMessageNoCompactSummary mirrors compact.ts failure when assistant text is empty after stream.
const ErrorMessageNoCompactSummary = "Failed to generate conversation summary - response did not contain valid text content"

// APIErrorMessagePrefix mirrors errors.ts API_ERROR_MESSAGE_PREFIX.
const APIErrorMessagePrefix = "API Error"

// CompactToolUseDenyMessage mirrors createCompactCanUseTool (compact.ts).
const CompactToolUseDenyMessage = "Tool use is not allowed during compaction"

// SkillTruncationMarker mirrors compact.ts SKILL_TRUNCATION_MARKER.
const SkillTruncationMarker = "\n\n[... skill content truncated for compaction; use Read on the skill path if you need the full text]"

// PTLRetryMarker mirrors compact.ts PTL_RETRY_MARKER (truncateHeadForPTLRetry user preamble).
const PTLRetryMarker = "[earlier conversation truncated for compaction retry]"

// Error strings exported for executor parity (compact.ts).
const (
	ErrorMessageNotEnoughMessages  = "Not enough messages to compact."
	ErrorMessagePromptTooLong      = "Conversation too long. Press esc twice to go up a few messages and try again."
	ErrorMessageUserAbort          = "API Error: Request was aborted."
	ErrorMessageIncompleteResponse = "Compaction interrupted · This may be due to network issues — please try again."
)

// MergeHookInstructions mirrors compact.ts mergeHookInstructions (empty input normalizes away).
// EffectiveCompactSummaryMaxTokens returns min(CompactSummaryMaxOutputTokens, defaultMax) when defaultMax > 0, else CompactSummaryMaxOutputTokens.
func EffectiveCompactSummaryMaxTokens(defaultMax int) int {
	if defaultMax > 0 && defaultMax < CompactSummaryMaxOutputTokens {
		return defaultMax
	}
	return CompactSummaryMaxOutputTokens
}

func MergeHookInstructions(userInstructions, hookInstructions string) string {
	u := strings.TrimSpace(userInstructions)
	h := strings.TrimSpace(hookInstructions)
	if h == "" {
		return userInstructions
	}
	if u == "" {
		return hookInstructions
	}
	return userInstructions + "\n\n" + hookInstructions
}

// StripImagesFromAPIMessagesJSON mirrors compact.ts stripImagesFromMessages for Anthropic Messages API JSON
// ([{ "role":"user"|"assistant", "content": ... }, ...]).
func StripImagesFromAPIMessagesJSON(transcript []byte) ([]byte, error) {
	var arr []interface{}
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return nil, err
	}
	changed := false
	for i, e := range arr {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role != "user" {
			continue
		}
		raw, ok := m["content"]
		if !ok {
			continue
		}
		blocks, ok := raw.([]interface{})
		if !ok {
			continue
		}
		newBlocks, ch := stripImagesInAPIContentBlocks(blocks)
		if !ch {
			continue
		}
		changed = true
		nm := cloneJSONObjMap(m)
		nm["content"] = newBlocks
		arr[i] = nm
	}
	if !changed {
		return append([]byte(nil), transcript...), nil
	}
	return json.Marshal(arr)
}

func stripImagesInAPIContentBlocks(blocks []interface{}) ([]interface{}, bool) {
	out := make([]interface{}, 0, len(blocks))
	changed := false
	for _, b := range blocks {
		bm, ok := b.(map[string]interface{})
		if !ok {
			out = append(out, b)
			continue
		}
		t, _ := bm["type"].(string)
		switch t {
		case "image":
			changed = true
			out = append(out, map[string]interface{}{"type": "text", "text": "[image]"})
		case "document":
			changed = true
			out = append(out, map[string]interface{}{"type": "text", "text": "[document]"})
		case "tool_result":
			raw, ok := bm["content"]
			if !ok {
				out = append(out, bm)
				continue
			}
			nested, ok := raw.([]interface{})
			if !ok {
				out = append(out, bm)
				continue
			}
			inner, ch := stripNestedMediaInToolResultContent(nested)
			if !ch {
				out = append(out, bm)
				continue
			}
			changed = true
			nb := cloneJSONObjMap(bm)
			nb["content"] = inner
			out = append(out, nb)
		default:
			out = append(out, bm)
		}
	}
	return out, changed
}

func stripNestedMediaInToolResultContent(items []interface{}) ([]interface{}, bool) {
	out := make([]interface{}, 0, len(items))
	changed := false
	for _, it := range items {
		im, ok := it.(map[string]interface{})
		if !ok {
			out = append(out, it)
			continue
		}
		t, _ := im["type"].(string)
		switch t {
		case "image":
			changed = true
			out = append(out, map[string]interface{}{"type": "text", "text": "[image]"})
		case "document":
			changed = true
			out = append(out, map[string]interface{}{"type": "text", "text": "[document]"})
		default:
			out = append(out, it)
		}
	}
	return out, changed
}

func cloneJSONObjMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// StripReinjectedAttachmentsFromTranscriptJSON mirrors compact.ts stripReinjectedAttachments for top-level
// transcript elements with type "attachment" (skill_discovery / skill_listing). No-op unless
// features.ExperimentalSkillSearchEnabled().
func StripReinjectedAttachmentsFromTranscriptJSON(transcript []byte) ([]byte, error) {
	if !features.ExperimentalSkillSearchEnabled() {
		return append([]byte(nil), transcript...), nil
	}
	var arr []interface{}
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return nil, err
	}
	out := make([]interface{}, 0, len(arr))
	changed := false
	for _, e := range arr {
		m, ok := e.(map[string]interface{})
		if ok && shouldDropReinjectedAttachmentMessage(m) {
			changed = true
			continue
		}
		out = append(out, e)
	}
	if !changed {
		return append([]byte(nil), transcript...), nil
	}
	return json.Marshal(out)
}

func shouldDropReinjectedAttachmentMessage(m map[string]interface{}) bool {
	typ, _ := m["type"].(string)
	if typ != "attachment" {
		return false
	}
	am, _ := m["attachment"].(map[string]interface{})
	if am == nil {
		return false
	}
	at, _ := am["type"].(string)
	return at == "skill_discovery" || at == "skill_listing"
}

// BuildPostCompactMessagesJSON mirrors compact.ts buildPostCompactMessages order:
// boundary, summaries, messagesToKeep, attachments, hookResults — each segment is a full message object JSON.
func BuildPostCompactMessagesJSON(boundary json.RawMessage, summaryMessages, messagesToKeep, attachments, hookResults []json.RawMessage) ([]byte, error) {
	var parts []json.RawMessage
	if len(bytes.TrimSpace(boundary)) > 0 {
		parts = append(parts, boundary)
	}
	parts = append(parts, summaryMessages...)
	parts = append(parts, messagesToKeep...)
	parts = append(parts, attachments...)
	parts = append(parts, hookResults...)
	if len(parts) == 0 {
		return []byte("[]"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	for _, p := range parts {
		if len(bytes.TrimSpace(p)) == 0 {
			continue
		}
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.Write(bytes.TrimSpace(p))
	}
	buf.WriteByte(']')
	if buf.Len() == 2 {
		return []byte("[]"), nil
	}
	return buf.Bytes(), nil
}

// AnnotateBoundaryWithPreservedSegmentJSON mirrors compact.ts annotateBoundaryWithPreservedSegment on compactMetadata.preservedSegment.
func AnnotateBoundaryWithPreservedSegmentJSON(boundary json.RawMessage, anchorUUID, headUUID, tailUUID string) (json.RawMessage, error) {
	h := strings.TrimSpace(headUUID)
	tail := strings.TrimSpace(tailUUID)
	if h == "" || tail == "" {
		if len(bytes.TrimSpace(boundary)) == 0 {
			return nil, nil
		}
		return append(json.RawMessage(nil), boundary...), nil
	}
	if len(bytes.TrimSpace(boundary)) == 0 {
		return nil, fmt.Errorf("compact: empty boundary with non-empty preserved segment")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(boundary, &m); err != nil {
		return nil, err
	}
	var cm map[string]interface{}
	if raw, ok := m["compactMetadata"].(map[string]interface{}); ok && raw != nil {
		cm = raw
	} else {
		cm = make(map[string]interface{})
		m["compactMetadata"] = cm
	}
	cm["preservedSegment"] = map[string]interface{}{
		"headUuid":   h,
		"anchorUuid": strings.TrimSpace(anchorUUID),
		"tailUuid":   tail,
	}
	return json.Marshal(m)
}

// StartsWithAPIErrorPrefix mirrors errors.ts startsWithApiErrorPrefix.
func StartsWithAPIErrorPrefix(text string) bool {
	if strings.HasPrefix(text, APIErrorMessagePrefix) {
		return true
	}
	return strings.HasPrefix(text, "Please run /login · "+APIErrorMessagePrefix)
}

// TruncateSkillContentRoughTokens mirrors compact.ts truncateToTokens (head + marker when over budget).
func TruncateSkillContentRoughTokens(content string, maxTokens int) string {
	if maxTokens <= 0 {
		return SkillTruncationMarker
	}
	if RoughTokenCountEstimationBytes(content) <= maxTokens {
		return content
	}
	charBudget := maxTokens*4 - len(SkillTruncationMarker)
	if charBudget < 1 {
		charBudget = 1
	}
	if charBudget > len(content) {
		charBudget = len(content)
	}
	return content[:charBudget] + SkillTruncationMarker
}

// PromptTooLongTokenGapFromAssistantJSON mirrors getPromptTooLongTokenGap (errors.ts + compact.ts):
// parses errorDetails or first assistant text block for "prompt is too long: … tokens > …".
func PromptTooLongTokenGapFromAssistantJSON(assistant []byte) (gap int64, ok bool) {
	var m map[string]interface{}
	if err := json.Unmarshal(assistant, &m); err != nil {
		return 0, false
	}
	detail, _ := m["errorDetails"].(string)
	if detail == "" {
		detail = firstAssistantTextFromContent(m["content"])
	}
	actual, limit, parsed := anthropic.ParsePromptTooLongTokenCounts(detail)
	if !parsed {
		return 0, false
	}
	g := actual - limit
	if g <= 0 {
		return 0, false
	}
	return g, true
}

func firstAssistantTextFromContent(content interface{}) string {
	for _, block := range contentBlocksGeneric(content) {
		bm, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		if typ, _ := bm["type"].(string); typ != "text" {
			continue
		}
		if t, ok := bm["text"].(string); ok {
			return t
		}
	}
	return ""
}

// messageLineAssistantRoundID returns whether the line is an assistant turn and the round id (message.id or top-level id).
func messageLineAssistantRoundID(raw json.RawMessage) (isAssistant bool, roundID string) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return false, ""
	}
	typ := lineTypeOrRole(m)
	if typ != "assistant" {
		return false, ""
	}
	if msg, ok := m["message"].(map[string]interface{}); ok {
		if s, ok := msg["id"].(string); ok && s != "" {
			return true, s
		}
	}
	if s, ok := m["id"].(string); ok {
		return true, s
	}
	return true, ""
}

func lineTypeOrRole(m map[string]interface{}) string {
	if t, ok := m["type"].(string); ok && t != "" {
		return t
	}
	if r, ok := m["role"].(string); ok {
		return r
	}
	return ""
}

// GroupRawMessagesByAPIRound mirrors grouping.ts groupMessagesByApiRound on a JSON transcript line array
// (supports internal {type,message.id} and Messages API {role,id} assistant lines).
func GroupRawMessagesByAPIRound(lines []json.RawMessage) [][]json.RawMessage {
	var groups [][]json.RawMessage
	var current []json.RawMessage
	var lastID *string

	for _, line := range lines {
		isAsst, id := messageLineAssistantRoundID(line)
		if isAsst && len(current) > 0 {
			same := lastID != nil && id == *lastID
			if !same {
				groups = append(groups, current)
				current = []json.RawMessage{line}
			} else {
				current = append(current, line)
			}
		} else {
			current = append(current, line)
		}
		if isAsst {
			lastID = new(string)
			*lastID = id
		}
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

func stripLeadingPTLRetryMarkerLines(lines []json.RawMessage) []json.RawMessage {
	if len(lines) == 0 || !isPTLRetryMetaLine(lines[0]) {
		return lines
	}
	return append([]json.RawMessage(nil), lines[1:]...)
}

func isPTLRetryMetaLine(raw json.RawMessage) bool {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	if t, ok := m["type"].(string); ok && t == "user" {
		if meta, ok := m["isMeta"].(bool); ok && meta {
			return userLinePlainContent(m) == PTLRetryMarker
		}
	}
	if r, ok := m["role"].(string); ok && r == "user" {
		return userLinePlainContent(m) == PTLRetryMarker
	}
	return false
}

func userLinePlainContent(m map[string]interface{}) string {
	c := m["content"]
	if s, ok := c.(string); ok {
		return s
	}
	arr, ok := c.([]interface{})
	if !ok || len(arr) != 1 {
		return ""
	}
	bm, ok := arr[0].(map[string]interface{})
	if !ok {
		return ""
	}
	if typ, _ := bm["type"].(string); typ != "text" {
		return ""
	}
	s, _ := bm["text"].(string)
	return s
}

func roughTokensForRawMessageGroup(group []json.RawMessage) int {
	if len(group) == 0 {
		return 0
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, r := range group {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(bytes.TrimSpace(r))
	}
	buf.WriteByte(']')
	n, err := EstimateMessageTokensFromAPIMessagesJSON(buf.Bytes())
	if err != nil {
		return 0
	}
	return n
}

// TruncateHeadForPTLRetryTranscriptJSON mirrors compact.ts truncateHeadForPTLRetry for Messages-style JSON arrays.
// assistant is the synthetic assistant payload carrying prompt-too-long text (for token gap); ok is false when
// nothing can be dropped without emptying the summarize set.
func TruncateHeadForPTLRetryTranscriptJSON(messages []byte, assistant []byte) ([]byte, bool) {
	var lines []json.RawMessage
	if err := json.Unmarshal(messages, &lines); err != nil || len(lines) == 0 {
		return nil, false
	}
	lines = stripLeadingPTLRetryMarkerLines(lines)
	groups := GroupRawMessagesByAPIRound(lines)
	if len(groups) < 2 {
		return nil, false
	}

	var dropCount int
	if gap, ok := PromptTooLongTokenGapFromAssistantJSON(assistant); ok {
		var acc int64
		dropCount = 0
		for _, g := range groups {
			acc += int64(roughTokensForRawMessageGroup(g))
			dropCount++
			if acc >= gap {
				break
			}
		}
	} else {
		dropCount = max(1, (len(groups)*20)/100)
	}

	dropCount = min(dropCount, len(groups)-1)
	if dropCount < 1 {
		return nil, false
	}

	var flat []json.RawMessage
	for _, g := range groups[dropCount:] {
		flat = append(flat, g...)
	}
	if len(flat) == 0 {
		return nil, false
	}

	firstAsst := false
	if typ := lineTypeOrRoleFromRaw(flat[0]); typ == "assistant" {
		firstAsst = true
	}
	if firstAsst {
		prefix, err := json.Marshal(map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]string{"type": "text", "text": PTLRetryMarker},
			},
		})
		if err != nil {
			return nil, false
		}
		flat = append([]json.RawMessage{prefix}, flat...)
	}
	out, err := json.Marshal(flat)
	if err != nil {
		return nil, false
	}
	return out, true
}

func lineTypeOrRoleFromRaw(raw json.RawMessage) string {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return lineTypeOrRole(m)
}

func contentBlocksGeneric(content interface{}) []interface{} {
	arr, ok := content.([]interface{})
	if ok {
		return arr
	}
	return nil
}

// CollectReadToolFilePathsFromTranscriptJSON mirrors compact.ts collectReadToolFilePaths (Messages API JSON array).
// Paths are filepath.Clean'd; expandPath parity is left to callers with real cwd.
func CollectReadToolFilePathsFromTranscriptJSON(transcript []byte) (map[string]struct{}, error) {
	var top []interface{}
	if err := json.Unmarshal(transcript, &top); err != nil {
		return nil, err
	}
	stubIDs := collectStubToolUseIDsFromTranscript(top)
	out := make(map[string]struct{})
	for _, e := range top {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if lineTypeOrRole(m) != "assistant" {
			continue
		}
		for _, block := range contentBlocksGeneric(m["content"]) {
			bm, ok := block.(map[string]interface{})
			if !ok {
				continue
			}
			if typ, _ := bm["type"].(string); typ != "tool_use" {
				continue
			}
			id, _ := bm["id"].(string)
			if stubIDs[id] {
				continue
			}
			name, _ := bm["name"].(string)
			if name != filereadtool.FileReadToolName {
				continue
			}
			fp := readToolInputFilePath(bm["input"])
			if fp == "" {
				continue
			}
			out[filepath.Clean(fp)] = struct{}{}
		}
	}
	return out, nil
}

func collectStubToolUseIDsFromTranscript(top []interface{}) map[string]bool {
	out := make(map[string]bool)
	for _, e := range top {
		m, ok := e.(map[string]interface{})
		if !ok || lineTypeOrRole(m) != "user" {
			continue
		}
		for _, block := range contentBlocksGeneric(m["content"]) {
			bm, ok := block.(map[string]interface{})
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
			if strings.HasPrefix(toolResultContentToString(bm["content"]), filereadtool.FileUnchangedStub) {
				out[tid] = true
			}
		}
	}
	return out
}

func toolResultContentToString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []interface{}:
		var b strings.Builder
		for _, it := range x {
			if m, ok := it.(map[string]interface{}); ok {
				if typ, _ := m["type"].(string); typ == "text" {
					if t, ok := m["text"].(string); ok {
						b.WriteString(t)
					}
				}
			}
		}
		return b.String()
	default:
		return ""
	}
}

// ToolInputFilePathFromJSON extracts Read tool file_path from tool_use input JSON (Messages API).
func ToolInputFilePathFromJSON(raw []byte) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return readToolInputFilePath(m)
}

func readToolInputFilePath(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case map[string]interface{}:
		s, _ := x["file_path"].(string)
		return s
	case string:
		var m map[string]interface{}
		if json.Unmarshal([]byte(x), &m) == nil {
			s, _ := m["file_path"].(string)
			return s
		}
		return ""
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		var m map[string]interface{}
		if json.Unmarshal(b, &m) != nil {
			return ""
		}
		s, _ := m["file_path"].(string)
		return s
	}
}
