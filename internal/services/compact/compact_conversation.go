package compact

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
)

// --- compact.ts orchestration helpers (pure JSON / no query import). Streaming, hooks, fork, attachments stay in engine/app. ---

var (
	// ErrPartialCompactNothingBefore mirrors partialCompactConversation (direction up_to, empty summarize slice).
	ErrPartialCompactNothingBefore = errors.New("Nothing to summarize before the selected message.")
	// ErrPartialCompactNothingAfter mirrors partialCompactConversation (direction from, empty summarize slice).
	ErrPartialCompactNothingAfter = errors.New("Nothing to summarize after the selected message.")
)

// CompactionResult mirrors compact.ts CompactionResult (logical shape; engine maps to persistence / events).
type CompactionResult struct {
	BoundaryMarkerJSON        json.RawMessage
	SummaryMessagesJSON       []json.RawMessage
	AttachmentsJSON           []json.RawMessage
	HookResultsJSON           []json.RawMessage
	MessagesToKeepJSON        []json.RawMessage
	UserDisplayMessage        string
	PreCompactTokenCount      int
	PostCompactTokenCount     int
	TruePostCompactTokenCount int
}

// AfterCompactBoundaryOptions mirrors getMessagesAfterCompactBoundary options (HISTORY_SNIP analogue).
type AfterCompactBoundaryOptions struct {
	IncludeSnipped bool
}

// IsCompactBoundaryMessageMap mirrors isCompactBoundaryMessage for a decoded message object.
func IsCompactBoundaryMessageMap(m map[string]interface{}) bool {
	if m == nil {
		return false
	}
	typ, _ := m["type"].(string)
	if typ != "system" {
		return false
	}
	st, _ := m["subtype"].(string)
	return st == "compact_boundary"
}

// FindLastCompactBoundaryIndexTranscriptJSON mirrors findLastCompactBoundaryIndex (-1 if none).
func FindLastCompactBoundaryIndexTranscriptJSON(transcript []byte) (int, error) {
	var lines []json.RawMessage
	if err := json.Unmarshal(transcript, &lines); err != nil {
		return 0, err
	}
	for i := len(lines) - 1; i >= 0; i-- {
		var m map[string]interface{}
		if err := json.Unmarshal(lines[i], &m); err != nil {
			continue
		}
		if IsCompactBoundaryMessageMap(m) {
			return i, nil
		}
	}
	return -1, nil
}

// GetMessagesAfterCompactBoundaryTranscriptJSON mirrors getMessagesAfterCompactBoundary (snip: drops top-level snipped:true when HISTORY_SNIP on).
func GetMessagesAfterCompactBoundaryTranscriptJSON(transcript []byte, opt AfterCompactBoundaryOptions) ([]byte, error) {
	idx, err := FindLastCompactBoundaryIndexTranscriptJSON(transcript)
	if err != nil {
		return nil, err
	}
	var lines []json.RawMessage
	if err := json.Unmarshal(transcript, &lines); err != nil {
		return nil, err
	}
	var slice []json.RawMessage
	if idx < 0 {
		slice = lines
	} else {
		slice = append([]json.RawMessage(nil), lines[idx:]...)
	}
	if features.HistorySnipEnabled() && !opt.IncludeSnipped {
		slice = filterSnippedTranscriptLines(slice)
	}
	if len(slice) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(slice)
}

func filterSnippedTranscriptLines(in []json.RawMessage) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(in))
	for _, raw := range in {
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) != nil {
			out = append(out, raw)
			continue
		}
		if sn, ok := m["snipped"].(bool); ok && sn {
			continue
		}
		out = append(out, raw)
	}
	return out
}

func isProgressMessageMap(m map[string]interface{}) bool {
	t, _ := m["type"].(string)
	return t == "progress"
}

func isCompactSummaryUserMap(m map[string]interface{}) bool {
	t, _ := m["type"].(string)
	if t != "user" {
		return false
	}
	b, ok := m["isCompactSummary"].(bool)
	return ok && b
}

func filterPartialKeepLines(lines []json.RawMessage, upTo bool) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(lines))
	for _, raw := range lines {
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) != nil {
			out = append(out, raw)
			continue
		}
		if isProgressMessageMap(m) {
			continue
		}
		if upTo {
			if IsCompactBoundaryMessageMap(m) || isCompactSummaryUserMap(m) {
				continue
			}
		}
		out = append(out, raw)
	}
	return out
}

// PartialCompactPartitionTranscriptJSON mirrors partialCompactConversation slicing (internal / hybrid transcript lines).
func PartialCompactPartitionTranscriptJSON(all []byte, pivot int, direction PartialCompactDirection) (summarize []byte, keep []byte, err error) {
	if direction == "" {
		direction = PartialCompactFrom
	}
	var lines []json.RawMessage
	if err := json.Unmarshal(all, &lines); err != nil {
		return nil, nil, err
	}
	n := len(lines)
	if pivot < 0 || pivot > n {
		return nil, nil, fmt.Errorf("compact: partial pivot %d out of range (len=%d)", pivot, n)
	}
	var sum, k []json.RawMessage
	switch direction {
	case PartialCompactUpTo:
		sum = lines[:pivot]
		k = filterPartialKeepLines(lines[pivot:], true)
	case PartialCompactFrom:
		sum = lines[pivot:]
		k = filterPartialKeepLines(lines[:pivot], false)
	default:
		return nil, nil, fmt.Errorf("compact: unknown partial direction %q", direction)
	}
	if len(sum) == 0 {
		if direction == PartialCompactUpTo {
			return nil, nil, ErrPartialCompactNothingBefore
		}
		return nil, nil, ErrPartialCompactNothingAfter
	}
	sbj, err1 := json.Marshal(sum)
	kbj, err2 := json.Marshal(k)
	if err1 != nil {
		return nil, nil, err1
	}
	if err2 != nil {
		return nil, nil, err2
	}
	return sbj, kbj, nil
}

// SelectPartialCompactAPIMessagesTranscriptJSON mirrors partial compact apiMessages selection (up_to → summarize only; from → full transcript).
func SelectPartialCompactAPIMessagesTranscriptJSON(all []byte, pivot int, direction PartialCompactDirection) ([]byte, error) {
	if direction == "" {
		direction = PartialCompactFrom
	}
	summarize, _, err := PartialCompactPartitionTranscriptJSON(all, pivot, direction)
	if err != nil {
		return nil, err
	}
	if direction == PartialCompactUpTo {
		return summarize, nil
	}
	return append([]byte(nil), all...), nil
}

// randomUUIDv4 is a compact RFC-4122 v4 string (no extra module deps).
func randomUUIDv4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// CreateCompactBoundaryMessageJSON mirrors createCompactBoundaryMessage (internal transcript shape).
func CreateCompactBoundaryMessageJSON(trigger string, preTokens int, lastPreCompactMessageUUID string, userContext string, messagesSummarized int) (json.RawMessage, error) {
	if trigger != "manual" && trigger != "auto" {
		return nil, fmt.Errorf("compact: boundary trigger must be manual or auto, got %q", trigger)
	}
	cm := map[string]interface{}{
		"trigger":   trigger,
		"preTokens": preTokens,
	}
	if strings.TrimSpace(userContext) != "" {
		cm["userContext"] = userContext
	}
	if messagesSummarized > 0 {
		cm["messagesSummarized"] = messagesSummarized
	}
	msg := map[string]interface{}{
		"type":            "system",
		"subtype":         "compact_boundary",
		"content":         "Conversation compacted",
		"isMeta":          false,
		"timestamp":       time.Now().UTC().Format(time.RFC3339Nano),
		"uuid":            randomUUIDv4(),
		"level":           "info",
		"compactMetadata": cm,
	}
	if id := strings.TrimSpace(lastPreCompactMessageUUID); id != "" {
		msg["logicalParentUuid"] = id
	}
	return json.Marshal(msg)
}

// AttachPreCompactDiscoveredToolsToBoundaryJSON sets compactMetadata.preCompactDiscoveredTools (sorted, like compact.ts).
func AttachPreCompactDiscoveredToolsToBoundaryJSON(boundary json.RawMessage, toolNames []string) (json.RawMessage, error) {
	if len(bytes.TrimSpace(boundary)) == 0 {
		return nil, errors.New("compact: empty boundary")
	}
	cp := append([]string(nil), toolNames...)
	sort.Strings(cp)
	seen := make(map[string]struct{})
	names := make([]interface{}, 0, len(cp))
	for _, n := range cp {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		names = append(names, n)
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
	if len(names) == 0 {
		cm["preCompactDiscoveredTools"] = nil
	} else {
		cm["preCompactDiscoveredTools"] = names
	}
	return json.Marshal(m)
}

// ExtractDiscoveredToolNamesFromTranscriptJSON mirrors extractDiscoveredToolNames (compact boundary carry + tool_reference in user tool_results).
func ExtractDiscoveredToolNamesFromTranscriptJSON(transcript []byte) ([]string, error) {
	var top []interface{}
	if err := json.Unmarshal(transcript, &top); err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, e := range top {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if IsCompactBoundaryMessageMap(m) {
			if cm, ok := m["compactMetadata"].(map[string]interface{}); ok {
				if arr, ok := cm["preCompactDiscoveredTools"].([]interface{}); ok {
					for _, x := range arr {
						if s, ok := x.(string); ok && s != "" {
							set[s] = struct{}{}
						}
					}
				}
			}
			continue
		}
		if !isUserLineMap(m) {
			continue
		}
		c := userOrAssistantContentField(m)
		accumulateToolReferencesFromUserContent(c, set)
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func isUserLineMap(m map[string]interface{}) bool {
	if t, ok := m["type"].(string); ok && t == "user" {
		return true
	}
	if r, ok := m["role"].(string); ok && r == "user" {
		return true
	}
	return false
}

func userOrAssistantContentField(m map[string]interface{}) interface{} {
	if msg, ok := m["message"].(map[string]interface{}); ok {
		if c, ok := msg["content"]; ok {
			return c
		}
	}
	return m["content"]
}

func accumulateToolReferencesFromUserContent(content interface{}, into map[string]struct{}) {
	arr, ok := content.([]interface{})
	if !ok {
		return
	}
	for _, b := range arr {
		bm, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		if typ, _ := bm["type"].(string); typ != "tool_result" {
			continue
		}
		walkToolReferenceItems(bm["content"], into)
	}
}

func walkToolReferenceItems(v interface{}, into map[string]struct{}) {
	switch x := v.(type) {
	case []interface{}:
		for _, it := range x {
			im, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			if typ, _ := im["type"].(string); typ == "tool_reference" {
				if n, ok := im["tool_name"].(string); ok && n != "" {
					into[n] = struct{}{}
				}
			}
			walkToolReferenceItems(im["content"], into)
		}
	default:
		return
	}
}

// CreateUserTextMessageJSON builds one Messages-API style user message (compact.ts createUserMessage with text content).
func CreateUserTextMessageJSON(text string) (json.RawMessage, error) {
	msg := map[string]interface{}{
		"role": "user",
		"content": []interface{}{
			map[string]string{"type": "text", "text": text},
		},
	}
	return json.Marshal(msg)
}

// AppendTranscriptMessagesJSON appends message objects to a transcript JSON array.
func AppendTranscriptMessagesJSON(transcript []byte, extra ...json.RawMessage) ([]byte, error) {
	var arr []json.RawMessage
	raw := bytes.TrimSpace(transcript)
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, err
		}
	}
	arr = append(arr, extra...)
	return json.Marshal(arr)
}

// BuildPartialCompactStreamRequestMessagesJSON mirrors partialCompactConversation stream request prefix:
// SelectPartialCompactAPIMessagesTranscriptJSON → stripImages → stripReinjectedAttachments → GetPartialCompactPrompt user.
func BuildPartialCompactStreamRequestMessagesJSON(fullTranscript []byte, pivot int, direction PartialCompactDirection, customInstructions string) ([]byte, error) {
	part, err := SelectPartialCompactAPIMessagesTranscriptJSON(fullTranscript, pivot, direction)
	if err != nil {
		return nil, err
	}
	tail, err := StripImagesFromAPIMessagesJSON(part)
	if err != nil {
		return nil, err
	}
	tail, err = StripReinjectedAttachmentsFromTranscriptJSON(tail)
	if err != nil {
		return nil, err
	}
	prompt := GetPartialCompactPrompt(customInstructions, direction)
	sum, err := CreateUserTextMessageJSON(prompt)
	if err != nil {
		return nil, err
	}
	return AppendTranscriptMessagesJSON(tail, sum)
}

// BuildCompactStreamRequestMessagesJSON mirrors streamCompactSummary normalize path prefix:
// getMessagesAfterCompactBoundary → stripImages → stripReinjectedAttachments → append summary user (Messages API JSON).
func BuildCompactStreamRequestMessagesJSON(transcript []byte, boundaryOpt AfterCompactBoundaryOptions, summaryPromptPlain string) ([]byte, error) {
	tail, err := GetMessagesAfterCompactBoundaryTranscriptJSON(transcript, boundaryOpt)
	if err != nil {
		return nil, err
	}
	tail, err = StripImagesFromAPIMessagesJSON(tail)
	if err != nil {
		return nil, err
	}
	tail, err = StripReinjectedAttachmentsFromTranscriptJSON(tail)
	if err != nil {
		return nil, err
	}
	sum, err := CreateUserTextMessageJSON(summaryPromptPlain)
	if err != nil {
		return nil, err
	}
	return AppendTranscriptMessagesJSON(tail, sum)
}

// CompactSummaryLooksLikePromptTooLong mirrors summary?.startsWith(PROMPT_TOO_LONG_ERROR_MESSAGE) in compact.ts.
func CompactSummaryLooksLikePromptTooLong(summaryText string) bool {
	return strings.HasPrefix(strings.TrimSpace(summaryText), anthropic.ErrPromptTooLongMessage)
}

// ShouldSuppressCompactErrorNotification mirrors addErrorNotificationIfNeeded exclusions (exact message match).
func ShouldSuppressCompactErrorNotification(errMsg string) bool {
	e := strings.TrimSpace(errMsg)
	return e == ErrorMessageUserAbort || e == ErrorMessageNotEnoughMessages
}

// LastTopLevelMessageUUIDTranscriptJSON returns the last transcript element's top-level "uuid" (internal message shape), or "".
func LastTopLevelMessageUUIDTranscriptJSON(transcript []byte) string {
	var lines []json.RawMessage
	if err := json.Unmarshal(transcript, &lines); err != nil {
		return ""
	}
	for i := len(lines) - 1; i >= 0; i-- {
		var m map[string]interface{}
		if err := json.Unmarshal(lines[i], &m); err != nil {
			continue
		}
		if u, ok := m["uuid"].(string); ok && strings.TrimSpace(u) != "" {
			return u
		}
	}
	return ""
}

// PostCompactTranscriptOptions configures BuildDefaultPostCompactTranscriptJSON (compact.ts buildPostCompactMessages subset).
type PostCompactTranscriptOptions struct {
	AutoCompact               bool
	SuppressFollowUpQuestions bool
	TranscriptPath            string
	LastPreCompactMessageUUID string
	// ExtraAttachmentsJSON is appended in buildPostCompactMessages order after summary, before hookResults (plan/skill/deltas from host).
	ExtraAttachmentsJSON   []json.RawMessage
	HookResultMessagesJSON []json.RawMessage
}

// BuildDefaultPostCompactTranscriptJSON builds [boundary, summaryUser, ...hookResults] (no messagesToKeep/attachments).
func BuildDefaultPostCompactTranscriptJSON(transcriptBeforeCompact []byte, rawAssistantSummary string, opt PostCompactTranscriptOptions) ([]byte, error) {
	lastUUID := strings.TrimSpace(opt.LastPreCompactMessageUUID)
	if lastUUID == "" {
		lastUUID = LastTopLevelMessageUUIDTranscriptJSON(transcriptBeforeCompact)
	}
	trigger := "manual"
	if opt.AutoCompact {
		trigger = "auto"
	}
	preTok, err := EstimateMessageTokensFromAPIMessagesJSON(transcriptBeforeCompact)
	if err != nil {
		preTok = 0
	}
	boundary, err := CreateCompactBoundaryMessageJSON(trigger, preTok, lastUUID, "", 0)
	if err != nil {
		return nil, err
	}
	names, err := ExtractDiscoveredToolNamesFromTranscriptJSON(transcriptBeforeCompact)
	if err != nil {
		return nil, err
	}
	boundary, err = AttachPreCompactDiscoveredToolsToBoundaryJSON(boundary, names)
	if err != nil {
		return nil, err
	}
	userSummaryText := GetCompactUserSummaryMessage(rawAssistantSummary, opt.SuppressFollowUpQuestions, opt.TranscriptPath, false)
	sumMsg, err := CreateUserTextMessageJSON(userSummaryText)
	if err != nil {
		return nil, err
	}
	return BuildPostCompactMessagesJSON(boundary, []json.RawMessage{sumMsg}, nil, opt.ExtraAttachmentsJSON, opt.HookResultMessagesJSON)
}
