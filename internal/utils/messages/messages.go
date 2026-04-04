// Package messages implements transcript I/O, API normalization, tool pairing, and
// a Go port of claude-code src/utils/messages.ts: messages.go holds disk + types.Message
// helpers; messages_ts_*.go holds TSMsg (map[string]any) parity for the TS Message union.
//
// Non-exhaustive map of TS exports: NormalizeMessages → NormalizeMessages; normalizeMessagesForAPI →
// NormalizeMessagesForAPI / NormalizeMessagesForAPIGeneric; mergeUserContentBlocks → MergeUserContentBlocks;
// mergeUserMessages → MergeUserMessagesMap; reorderAttachmentsForAPI → ReorderAttachmentsForAPIGeneric;
// stripPromptXMLTags → StripPromptXMLTags; ensureToolResultPairing → EnsureToolResultPairing;
// handleMessageFromStream → HandleMessageFromStream; buildMessageLookups → BuildMessageLookups.
//
// Feature gates use env vars where TS used GrowthBook/Statsig: RABBIT_AUTO_MEMORY,
// RABBIT_AMBER_PRISM, RABBIT_BASH_CLASSIFIER, RABBIT_TOOL_SEARCH, RABBIT_HISTORY_SNIP,
// RABBIT_KAIROS, RABBIT_KAIROS_CHANNELS, RABBIT_AGENT_SWARMS, RABBIT_EXPERIMENTAL_SKILL_SEARCH,
// RABBIT_TODO_V2, RABBIT_PLAN_MODE_INTERVIEW, RABBIT_STRICT_TOOL_PAIRING, RABBIT_TENGU_CHAIR_SERMON,
// RABBIT_TENGU_TOOLREF_DEFER, RABBIT_NON_INTERACTIVE (API error strip-target copy), RABBIT_CONNECTOR_TEXT,
// RABBIT_SNIP_RUNTIME_ENABLED (0 disables snip merge/[id:] runtime), RABBIT_SNIP_NUDGE_TEXT, RABBIT_KAIROS_BRIEF,
// RABBIT_FEATURE_KAIROS (Brief alias), RABBIT_ANT_UNKNOWN_ATTACHMENT (log unknown attachment types like TS logAntError),
// RABBIT_OUTPUT_STYLE_NAMES_JSON (inline style display map), RABBIT_OUTPUT_STYLE_CONFIG_PATH (JSON file like OUTPUT_STYLE_CONFIG),
// RABBIT_OUTPUT_STYLE_SCAN_DIRS (path list of dirs with *.md styles, like .claude/output-styles),
// RABBIT_OUTPUT_STYLE_PLUGINS_PATH / RABBIT_OUTPUT_STYLE_PLUGINS_JSON (JSON array of {plugin,dir} for recursive *.md, TS loadPluginOutputStyles),
// RABBIT_SETTINGS_OUTPUT_STYLE / RABBIT_CLAUDE_SETTINGS_PATH (settings outputStyle fallback when attachment.style is empty/default),
// RABBIT_BASH_MAX_OUTPUT_LENGTH (notebook text truncation / TS BASH_MAX_OUTPUT_LENGTH, default 30000 cap 150000),
// RABBIT_TASK_OUTPUT_DIR (Bash background output path when transcript omits backgroundTaskOutputPath),
// RABBIT_MCP_RESOURCE_DEBUG (log empty MCP resource render), etc.
package messages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/2456868764/rabbit-code/internal/types"
)

// --- Transcript (on-disk) ---------------------------------------------------

// CurrentTranscriptVersion is written to new transcript files.
const CurrentTranscriptVersion = 1

// Transcript is a versioned on-disk conversation (session file subset).
type Transcript struct {
	TranscriptVersion int             `json:"transcript_version"`
	Messages          []types.Message `json:"messages"`
}

// CanonicalJSON returns deterministic JSON for hashing and golden tests.
func CanonicalJSON(t *Transcript) ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf("transcript is nil")
	}
	return json.MarshalIndent(t, "", "  ")
}

// SHA256Hex returns hex-encoded SHA-256 of CanonicalJSON.
func SHA256Hex(t *Transcript) (string, error) {
	b, err := CanonicalJSON(t)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ParseTranscriptJSON decodes transcript JSON and sets default version if missing.
func ParseTranscriptJSON(data []byte) (*Transcript, error) {
	var tr Transcript
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, err
	}
	if tr.TranscriptVersion == 0 {
		tr.TranscriptVersion = CurrentTranscriptVersion
	}
	return &tr, nil
}

// ReadTranscriptFile reads and parses a transcript JSON file.
func ReadTranscriptFile(path string) (*Transcript, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseTranscriptJSON(b)
}

// WriteTranscriptFile writes canonical JSON atomically (same directory).
func WriteTranscriptFile(path string, t *Transcript) error {
	if t == nil {
		return fmt.Errorf("transcript is nil")
	}
	if t.TranscriptVersion == 0 {
		t.TranscriptVersion = CurrentTranscriptVersion
	}
	b, err := CanonicalJSON(t)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".transcript-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// --- NormalizeForAPI (types.Message) ----------------------------------------

// NormalizeOptions controls NormalizeForAPI.
type NormalizeOptions struct {
	StripNonAPI     bool
	ConnectorToText bool
}

// DefaultNormalizeAPI returns options suitable for sending to the Messages API.
func DefaultNormalizeAPI() NormalizeOptions {
	return NormalizeOptions{StripNonAPI: true, ConnectorToText: true}
}

// NormalizeForAPI returns a copy of messages with internal fields stripped or mapped for API use.
func NormalizeForAPI(msgs []types.Message, opt NormalizeOptions) []types.Message {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]types.Message, 0, len(msgs))
	for _, m := range msgs {
		if opt.StripNonAPI && m.Role == types.RoleProgress {
			continue
		}
		nm := types.Message{Role: m.Role, Content: normalizeContent(m.Content, opt)}
		if len(nm.Content) == 0 && opt.StripNonAPI {
			continue
		}
		out = append(out, nm)
	}
	return out
}

func normalizeContent(in []types.ContentPiece, opt NormalizeOptions) []types.ContentPiece {
	if len(in) == 0 {
		return nil
	}
	out := make([]types.ContentPiece, 0, len(in))
	for _, c := range in {
		switch c.Type {
		case types.BlockTypeText, types.BlockTypeToolUse, types.BlockTypeToolResult:
			out = append(out, c)
		case types.BlockTypeConnectorText:
			if opt.ConnectorToText {
				out = append(out, types.ContentPiece{Type: types.BlockTypeText, Text: c.Text})
			} else {
				out = append(out, c)
			}
		case types.BlockTypeFileRef:
			if !opt.StripNonAPI {
				out = append(out, c)
			} else {
				ref := strings.TrimSpace(c.Ref)
				if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
					if src, err := json.Marshal(map[string]string{"type": "url", "url": ref}); err == nil {
						out = append(out, types.ContentPiece{Type: types.BlockTypeDocument, Source: json.RawMessage(src)})
					}
				}
			}
		case types.BlockTypeBoundary, types.BlockTypeTombstone, types.BlockTypeHistorySnip,
			types.BlockTypeCompactionReminder, types.BlockTypeKairosQueue, types.BlockTypeKairosChannel,
			types.BlockTypeKairosBrief, types.BlockTypeUDSInbox, types.BlockTypeProgress:
			if !opt.StripNonAPI {
				out = append(out, c)
			}
		default:
			if !opt.StripNonAPI {
				out = append(out, c)
			}
		}
	}
	return out
}

// StripToolResultSignature is a no-op placeholder for signature stripping (parity with TS).
func StripToolResultSignature(raw json.RawMessage) json.RawMessage { return raw }

// --- File refs --------------------------------------------------------------

// VerifyFileRefs checks each file_ref content piece: path readable and optional Sha256 matches.
func VerifyFileRefs(msgs []types.Message) error {
	for mi, m := range msgs {
		for ci, c := range m.Content {
			if c.Type != types.BlockTypeFileRef {
				continue
			}
			if c.Ref == "" {
				return fmt.Errorf("message %d content %d: file_ref missing ref", mi, ci)
			}
			b, err := os.ReadFile(c.Ref)
			if err != nil {
				return fmt.Errorf("message %d content %d: file_ref %q: %w", mi, ci, c.Ref, err)
			}
			if c.Sha256 != "" {
				sum := sha256.Sum256(b)
				got := hex.EncodeToString(sum[:])
				if got != c.Sha256 {
					return fmt.Errorf("message %d content %d: file_ref %q sha256 want %s got %s", mi, ci, c.Ref, c.Sha256, got)
				}
			}
		}
	}
	return nil
}

// --- Tool pairing -----------------------------------------------------------

// ValidateToolPairing checks assistant tool_use blocks are followed by matching user tool_results.
func ValidateToolPairing(msgs []types.Message, strict bool) error {
	for i := range msgs {
		if msgs[i].Role != types.RoleAssistant {
			continue
		}
		need := toolUseIDs(msgs[i].Content)
		if len(need) == 0 {
			continue
		}
		if i+1 >= len(msgs) {
			if strict {
				return fmt.Errorf("message %d (assistant): tool_use ids %v but no following message for tool_result", i, need)
			}
			continue
		}
		next := msgs[i+1]
		if next.Role != types.RoleUser {
			if strict {
				return fmt.Errorf("message %d (assistant): expected next role user for tool_result, got %q", i, next.Role)
			}
			continue
		}
		got := toolResultIDs(next.Content)
		missing, extra := diffToolIDCounts(need, got)
		if len(missing) > 0 {
			return fmt.Errorf("message %d (assistant) / message %d (user) tool_result mismatch: missing tool_result for id(s) %v",
				i, i+1, missing)
		}
		if strict && len(extra) > 0 {
			return fmt.Errorf("message %d (assistant) / message %d (user) tool_result mismatch: unexpected tool_result id(s) %v",
				i, i+1, extra)
		}
	}
	return nil
}

func toolUseIDs(c []types.ContentPiece) []string {
	var ids []string
	for _, p := range c {
		if p.Type == types.BlockTypeToolUse && p.ID != "" {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func toolResultIDs(c []types.ContentPiece) []string {
	var ids []string
	for _, p := range c {
		if p.Type == types.BlockTypeToolResult && p.ToolUseID != "" {
			ids = append(ids, p.ToolUseID)
		}
	}
	return ids
}

func diffToolIDCounts(need, got []string) (missing, extra []string) {
	nc := countIDs(need)
	gc := countIDs(got)
	for id, n := range nc {
		g := gc[id]
		for i := 0; i < n-g; i++ {
			missing = append(missing, id)
		}
	}
	for id, g := range gc {
		n := nc[id]
		for i := 0; i < g-n; i++ {
			extra = append(extra, id)
		}
	}
	return missing, extra
}

func countIDs(ids []string) map[string]int {
	m := make(map[string]int)
	for _, id := range ids {
		m[id]++
	}
	return m
}

// --- History snip -----------------------------------------------------------

// StripHistorySnipPieces removes history_snip content blocks from each message.
func StripHistorySnipPieces(msgs []types.Message) []types.Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]types.Message, 0, len(msgs))
	for _, m := range msgs {
		var keep []types.ContentPiece
		for _, p := range m.Content {
			if p.Type == types.BlockTypeHistorySnip {
				continue
			}
			keep = append(keep, p)
		}
		if len(keep) == 0 {
			continue
		}
		m.Content = keep
		out = append(out, m)
	}
	return out
}

// --- messages.ts string exports & small helpers ----------------------------

const memoryCorrectionHint = "\n\nNote: The user's next message may contain a correction or preference. Pay close attention — if they explain what went wrong or how they'd prefer you to work, consider saving that to memory for future sessions."

// ToolReferenceTurnBoundary is injected beside tool_reference tool_results in TS normalizeMessagesForAPI.
const ToolReferenceTurnBoundary = "Tool loaded."

// WithMemoryCorrectionHint mirrors TS withMemoryCorrectionHint using env instead of GrowthBook:
// RABBIT_AUTO_MEMORY=1 and RABBIT_AMBER_PRISM=1 append the hint.
func WithMemoryCorrectionHint(message string) string {
	if os.Getenv("RABBIT_AUTO_MEMORY") == "1" && os.Getenv("RABBIT_AMBER_PRISM") == "1" {
		return message + memoryCorrectionHint
	}
	return message
}

// DeriveShortMessageId mirrors TS deriveShortMessageId (6-char base36 from UUID prefix).
func DeriveShortMessageId(uuid string) string {
	hex := strings.ReplaceAll(uuid, "-", "")
	if len(hex) > 10 {
		hex = hex[:10]
	}
	if hex == "" {
		return ""
	}
	n, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return ""
	}
	s := strconv.FormatUint(n, 36)
	if len(s) > 6 {
		return s[:6]
	}
	return s
}

const (
	InterruptMessage                      = "[Request interrupted by user]"
	InterruptMessageForToolUse            = "[Request interrupted by user for tool use]"
	CancelMessage                         = "The user doesn't want to take this action right now. STOP what you are doing and wait for the user to tell you how to proceed."
	RejectMessage                         = "The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). STOP what you are doing and wait for the user to tell you how to proceed."
	RejectMessageWithReasonPrefix         = "The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). To tell you how to proceed, the user said:\n"
	SubagentRejectMessage                 = "Permission for this tool use was denied. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). Try a different approach or report the limitation to complete your task."
	SubagentRejectMessageWithReasonPrefix = "Permission for this tool use was denied. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). The user said:\n"
	PlanRejectionPrefix                   = "The agent proposed a plan that was rejected by the user. The user chose to stay in plan mode rather than proceed with implementation.\n\nRejected plan:\n"
	NoResponseRequested                   = "No response requested."
	SyntheticToolResultPlaceholder        = "[Tool result missing due to internal error]"
	SyntheticModel                        = "<synthetic>"
)

// DenialWorkaroundGuidance mirrors TS DENIAL_WORKAROUND_GUIDANCE.
const DenialWorkaroundGuidance = `IMPORTANT: You *may* attempt to accomplish this action using other tools that might naturally be used to accomplish this goal, ` +
	`e.g. using head instead of cat. But you *should not* attempt to work around this denial in malicious ways, ` +
	`e.g. do not use your ability to run tests to execute non-test actions. ` +
	`You should only try to work around this restriction in reasonable ways that do not attempt to bypass the intent behind this denial. ` +
	`If you believe this capability is essential to complete the user's request, STOP and explain to the user ` +
	`what you were trying to do and why you need this permission. Let the user decide how to proceed.`

// AutoRejectMessage mirrors TS AUTO_REJECT_MESSAGE(toolName).
func AutoRejectMessage(toolName string) string {
	return fmt.Sprintf("Permission to use %s has been denied. %s", toolName, DenialWorkaroundGuidance)
}

// DontAskRejectMessage mirrors TS DONT_ASK_REJECT_MESSAGE(toolName).
func DontAskRejectMessage(toolName string) string {
	return fmt.Sprintf("Permission to use %s has been denied because Claude Code is running in don't ask mode. %s", toolName, DenialWorkaroundGuidance)
}

const autoModeRejectionPrefix = "Permission for this action has been denied. Reason: "

// IsClassifierDenial mirrors TS isClassifierDenial.
func IsClassifierDenial(content string) bool {
	return strings.HasPrefix(content, autoModeRejectionPrefix)
}

// BuildYoloRejectionMessage mirrors TS buildYoloRejectionMessage (Bash classifier hint via RABBIT_BASH_CLASSIFIER=1).
func BuildYoloRejectionMessage(reason string) string {
	ruleHint := `To allow this type of action in the future, the user can add a Bash permission rule to their settings.`
	if os.Getenv("RABBIT_BASH_CLASSIFIER") == "1" {
		ruleHint = `To allow this type of action in the future, the user can add a permission rule like ` +
			`Bash(prompt: <description of allowed action>) to their settings. ` +
			`At the end of your session, recommend what permission rules to add so you don't get blocked again.`
	}
	return autoModeRejectionPrefix + reason + ". " +
		`If you have other tasks that don't depend on this action, continue working on those. ` +
		DenialWorkaroundGuidance + " " + ruleHint
}

// BuildClassifierUnavailableMessage mirrors TS buildClassifierUnavailableMessage.
func BuildClassifierUnavailableMessage(toolName, classifierModel string) string {
	return classifierModel + " is temporarily unavailable, so auto mode cannot determine the safety of " + toolName + " right now. " +
		`Wait briefly and then try this action again. ` +
		`If it keeps failing, continue with other tasks that don't require this action and come back to it later. ` +
		`Note: reading files, searching code, and other read-only operations do not require the classifier and can still be used.`
}

// SyntheticMessageTexts returns the set of first text-block bodies treated as synthetic in TS isSyntheticMessage.
func SyntheticMessageTexts() []string {
	return []string{
		InterruptMessage,
		InterruptMessageForToolUse,
		CancelMessage,
		RejectMessage,
		NoResponseRequested,
	}
}

// IsSyntheticMessageMap mirrors TS isSyntheticMessage for decoded map[string]any envelopes (type, message.content).
func IsSyntheticMessageMap(msg map[string]any) bool {
	t, _ := msg["type"].(string)
	if t == "progress" || t == "attachment" || t == "system" {
		return false
	}
	inner, ok := msg["message"].(map[string]any)
	if !ok {
		return false
	}
	content := inner["content"]
	arr, ok := content.([]any)
	if !ok || len(arr) == 0 {
		return false
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return false
	}
	if typ, _ := first["type"].(string); typ != "text" {
		return false
	}
	text, _ := first["text"].(string)
	for _, s := range SyntheticMessageTexts() {
		if text == s {
			return true
		}
	}
	return false
}

// WrapInSystemReminder mirrors TS wrapInSystemReminder.
func WrapInSystemReminder(content string) string {
	return "<system-reminder>\n" + content + "\n</system-reminder>"
}

// PLAN_PHASE4_CONTROL mirrors TS export PLAN_PHASE4_CONTROL.
const PLAN_PHASE4_CONTROL = `### Phase 4: Final Plan
Goal: Write your final plan to the plan file (the only file you can edit).
- Begin with a **Context** section: explain why this change is being made — the problem or need it addresses, what prompted it, and the intended outcome
- Include only your recommended approach, not all alternatives
- Ensure that the plan file is concise enough to scan quickly, but detailed enough to execute effectively
- Include the paths of critical files to be modified
- Reference existing functions and utilities you found that should be reused, with their file paths
- Include a verification section describing how to test the changes end-to-end (run the code, use MCP tools, run tests)`

// --- Generic JSON-shaped pipeline (TS Message[] subset) ---------------------

// ErrAttachmentNeedsNormalizer is returned when an attachment message is present but no expander was configured.
var ErrAttachmentNeedsNormalizer = errors.New("messages: attachment message requires NormalizeAttachment callback (TS normalizeAttachmentForAPI)")

// NormalizeMessagesForAPIConfig configures NormalizeMessagesForAPIGeneric (full TS normalizeMessagesForAPI parity).
type NormalizeMessagesForAPIConfig struct {
	ToolSearchEnabled bool
	// NormalizeAttachment converts one attachment envelope to zero or more user-shaped messages; required if input contains type "attachment".
	NormalizeAttachment func(att map[string]any) ([]map[string]any, error)
	// When ToolSearchEnabled, strip tool_reference blocks for tools not in this set (empty/nil → all references treated unavailable, matching TS empty Set).
	AvailableToolNames map[string]struct{}
	// Optional tool registry (name + aliases) for canonical names and normalizeToolInputForAPI; nil uses legacy aliases only.
	Tools []ToolSpec
	// Optional extra per-tool_use hook after canonical name + NormalizeToolInputForAPIMap.
	NormalizeToolUseBlock func(block map[string]any) map[string]any
}

// ReorderAttachmentsForAPIGeneric mirrors TS reorderAttachmentsForAPI for []map[string]any (json.Unmarshal transcript-style).
func ReorderAttachmentsForAPIGeneric(msgs []map[string]any) []map[string]any {
	if len(msgs) == 0 {
		return nil
	}
	var result []map[string]any
	var pending []map[string]any
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		t, _ := m["type"].(string)
		if t == "attachment" {
			pending = append(pending, m)
			continue
		}
		isStop := t == "assistant" || userMessageLeadingBlockIsToolResult(m)
		if isStop && len(pending) > 0 {
			result = append(result, pending...)
			result = append(result, m)
			pending = pending[:0]
		} else {
			result = append(result, m)
		}
	}
	result = append(result, pending...)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func userMessageLeadingBlockIsToolResult(m map[string]any) bool {
	t, _ := m["type"].(string)
	if t != "user" {
		return false
	}
	msg, ok := m["message"].(map[string]any)
	if !ok {
		return false
	}
	arr, ok := msg["content"].([]any)
	if !ok || len(arr) == 0 {
		return false
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return false
	}
	bt, _ := first["type"].(string)
	return bt == "tool_result"
}

// IsSystemLocalCommandMessageMap mirrors TS isSystemLocalCommandMessage.
func IsSystemLocalCommandMessageMap(m map[string]any) bool {
	t, _ := m["type"].(string)
	if t != "system" {
		return false
	}
	st, _ := m["subtype"].(string)
	return st == "local_command"
}

// StripToolReferenceBlocksFromUserMessageMap mirrors stripToolReferenceBlocksFromUserMessage for map-shaped content.
func StripToolReferenceBlocksFromUserMessageMap(msg map[string]any) map[string]any {
	out := cloneMapJSON(msg)
	inner, ok := out["message"].(map[string]any)
	if !ok {
		return out
	}
	content := inner["content"]
	arr, ok := content.([]any)
	if !ok {
		return out
	}
	newArr := make([]any, 0, len(arr))
	changed := false
	for _, item := range arr {
		block, ok := item.(map[string]any)
		if !ok {
			newArr = append(newArr, item)
			continue
		}
		bt, _ := block["type"].(string)
		if bt != "tool_result" {
			newArr = append(newArr, block)
			continue
		}
		innerC, ok := block["content"].([]any)
		if !ok {
			newArr = append(newArr, block)
			continue
		}
		filtered := make([]any, 0, len(innerC))
		for _, c := range innerC {
			cm, ok := c.(map[string]any)
			if !ok {
				filtered = append(filtered, c)
				continue
			}
			if typ, _ := cm["type"].(string); typ == "tool_reference" {
				changed = true
				continue
			}
			filtered = append(filtered, c)
		}
		if len(filtered) == 0 {
			changed = true
			nb := cloneMapJSON(block)
			nb["content"] = []any{map[string]any{"type": "text", "text": "[Tool references removed - tool search not enabled]"}}
			newArr = append(newArr, nb)
			continue
		}
		if len(filtered) != len(innerC) {
			changed = true
			nb := cloneMapJSON(block)
			nb["content"] = filtered
			newArr = append(newArr, nb)
			continue
		}
		newArr = append(newArr, block)
	}
	if !changed {
		return out
	}
	inner["content"] = newArr
	return out
}

// StripCallerFieldFromAssistantMessageMap mirrors stripCallerFieldFromAssistantMessage.
func StripCallerFieldFromAssistantMessageMap(msg map[string]any) map[string]any {
	out := cloneMapJSON(msg)
	inner, ok := out["message"].(map[string]any)
	if !ok {
		return out
	}
	content, ok := inner["content"].([]any)
	if !ok {
		return out
	}
	changed := false
	newContent := make([]any, 0, len(content))
	for _, item := range content {
		block, ok := item.(map[string]any)
		if !ok {
			newContent = append(newContent, item)
			continue
		}
		if typ, _ := block["type"].(string); typ != "tool_use" {
			newContent = append(newContent, block)
			continue
		}
		if _, has := block["caller"]; !has {
			newContent = append(newContent, block)
			continue
		}
		changed = true
		trim := map[string]any{
			"type":  "tool_use",
			"id":    block["id"],
			"name":  block["name"],
			"input": block["input"],
		}
		newContent = append(newContent, trim)
	}
	if !changed {
		return out
	}
	inner["content"] = newContent
	return out
}

// MergeAssistantMessagesMap mirrors mergeAssistantMessages.
func MergeAssistantMessagesMap(a, b map[string]any) map[string]any {
	out := cloneMapJSON(a)
	ma, ok := out["message"].(map[string]any)
	if !ok {
		return out
	}
	mb, ok := b["message"].(map[string]any)
	if !ok {
		return out
	}
	ca, _ := ma["content"].([]any)
	cb, _ := mb["content"].([]any)
	merged := append(append([]any{}, ca...), cb...)
	ma["content"] = merged
	return out
}

// MergeUserMessagesMap mirrors mergeUserMessages (HISTORY_SNIP + isSnipRuntimeEnabled isMeta semantics; uuid ternary matches TS).
func MergeUserMessagesMap(a, b map[string]any) map[string]any {
	out := cloneMapJSON(a)
	ma, ok := out["message"].(map[string]any)
	if !ok {
		return out
	}
	mb, ok := b["message"].(map[string]any)
	if !ok {
		return out
	}
	last := normalizeUserContentToBlocks(ma["content"])
	cur := normalizeUserContentToBlocks(mb["content"])
	joined := joinTextAtSeamBlocks(last, cur)
	ma["content"] = blocksToAnySlice(hoistToolResultsBlocks(joined))
	isMetaA := truthy(a["isMeta"])
	isMetaB := truthy(b["isMeta"])
	if IsSnipRuntimeEnabled() {
		if isMetaA && isMetaB {
			out["isMeta"] = true
		} else {
			delete(out, "isMeta")
		}
	}
	if isMetaA {
		if u, ok := b["uuid"].(string); ok && u != "" {
			out["uuid"] = u
		}
	}
	return out
}

// MergeUserMessagesAndToolResultsMap mirrors mergeUserMessagesAndToolResults (mergeUserContentBlocks + hoist).
func MergeUserMessagesAndToolResultsMap(a, b map[string]any) map[string]any {
	out := cloneMapJSON(a)
	ma, ok := out["message"].(map[string]any)
	if !ok {
		return out
	}
	mb, ok := b["message"].(map[string]any)
	if !ok {
		return out
	}
	last := normalizeUserContentToBlocks(ma["content"])
	cur := normalizeUserContentToBlocks(mb["content"])
	merged := MergeUserContentBlocksMap(last, cur, tenguChairSermonEnabled())
	ma["content"] = blocksToAnySlice(hoistToolResultsBlocks(merged))
	return out
}

func tenguChairSermonEnabled() bool {
	return os.Getenv("RABBIT_TENGU_CHAIR_SERMON") == "1"
}

// MergeUserContentBlocks mirrors TS mergeUserContentBlocks (honors RABBIT_TENGU_CHAIR_SERMON).
func MergeUserContentBlocks(a, b []map[string]any) []map[string]any {
	return MergeUserContentBlocksMap(a, b, tenguChairSermonEnabled())
}

// MergeUserContentBlocksMap mirrors TS mergeUserContentBlocks for []map[string]any content blocks.
func MergeUserContentBlocksMap(a, b []map[string]any, chairSermon bool) []map[string]any {
	if len(a) == 0 {
		return append([]map[string]any{}, b...)
	}
	lastA := a[len(a)-1]
	if mapStr(lastA, "type") != "tool_result" {
		return append(append([]map[string]any{}, a...), b...)
	}
	if !chairSermon {
		if _, ok := lastA["content"].(string); ok {
			allText := true
			for _, x := range b {
				if mapStr(x, "type") != "text" {
					allText = false
					break
				}
			}
			if allText {
				if smooshed := smooshIntoToolResultMap(lastA, b); smooshed != nil {
					out := append([]map[string]any{}, a[:len(a)-1]...)
					return append(out, smooshed)
				}
			}
		}
		return append(append([]map[string]any{}, a...), b...)
	}
	var toSmoosh, toolResults []map[string]any
	for _, x := range b {
		if mapStr(x, "type") == "tool_result" {
			toolResults = append(toolResults, x)
		} else {
			toSmoosh = append(toSmoosh, x)
		}
	}
	if len(toSmoosh) == 0 {
		return append(append([]map[string]any{}, a...), b...)
	}
	if smooshed := smooshIntoToolResultMap(lastA, toSmoosh); smooshed != nil {
		out := append(append([]map[string]any{}, a[:len(a)-1]...), smooshed)
		return append(out, toolResults...)
	}
	return append(append([]map[string]any{}, a...), b...)
}

func mapStr(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

func toolResultContentHasToolReference(content any) bool {
	arr, ok := content.([]any)
	if !ok {
		return false
	}
	for _, it := range arr {
		cm, ok := it.(map[string]any)
		if ok && mapStr(cm, "type") == "tool_reference" {
			return true
		}
	}
	return false
}

// smooshIntoToolResultMap mirrors TS smooshIntoToolResult; returns nil if smoosh impossible (tool_reference constraint).
func smooshIntoToolResultMap(tr map[string]any, blocks []map[string]any) map[string]any {
	if len(blocks) == 0 {
		return tr
	}
	out := cloneMapJSON(tr)
	existing := out["content"]
	if toolResultContentHasToolReference(existing) {
		return nil
	}
	if truthy(out["is_error"]) {
		var textOnly []map[string]any
		for _, b := range blocks {
			if mapStr(b, "type") == "text" {
				textOnly = append(textOnly, b)
			}
		}
		blocks = textOnly
		if len(blocks) == 0 {
			return out
		}
	}
	allText := true
	for _, b := range blocks {
		if mapStr(b, "type") != "text" {
			allText = false
			break
		}
	}
	if allText {
		if existing == nil {
			var parts []string
			for _, b := range blocks {
				parts = append(parts, strings.TrimSpace(mapStr(b, "text")))
			}
			out["content"] = strings.Join(filterNonEmpty(parts), "\n\n")
			return out
		}
		if s, ok := existing.(string); ok {
			parts := []string{strings.TrimSpace(s)}
			for _, b := range blocks {
				parts = append(parts, strings.TrimSpace(mapStr(b, "text")))
			}
			out["content"] = strings.Join(filterNonEmpty(parts), "\n\n")
			return out
		}
	}
	var base []map[string]any
	switch ex := existing.(type) {
	case nil:
		base = nil
	case string:
		if strings.TrimSpace(ex) != "" {
			base = []map[string]any{{"type": "text", "text": strings.TrimSpace(ex)}}
		}
	case []any:
		for _, it := range ex {
			if bm, ok := it.(map[string]any); ok {
				base = append(base, bm)
			}
		}
	default:
		base = nil
	}
	merged := mergeAdjacentTextBlocksInToolResult(append(base, blocks...))
	out["content"] = blocksToAnySlice(merged)
	return out
}

func filterNonEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func mergeAdjacentTextBlocksInToolResult(blocks []map[string]any) []map[string]any {
	var merged []map[string]any
	for _, b := range blocks {
		if mapStr(b, "type") != "text" {
			merged = append(merged, b)
			continue
		}
		t := strings.TrimSpace(mapStr(b, "text"))
		if t == "" {
			continue
		}
		if len(merged) > 0 {
			prev := merged[len(merged)-1]
			if mapStr(prev, "type") == "text" {
				pt := strings.TrimSpace(mapStr(prev, "text"))
				merged[len(merged)-1] = map[string]any{"type": "text", "text": pt + "\n\n" + t}
				continue
			}
		}
		merged = append(merged, map[string]any{"type": "text", "text": t})
	}
	return merged
}

// MergeAdjacentUserMessagesGeneric merges consecutive user messages in a mixed user|assistant slice.
func MergeAdjacentUserMessagesGeneric(msgs []map[string]any) []map[string]any {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		t, _ := m["type"].(string)
		if t == "user" && len(out) > 0 {
			prev := out[len(out)-1]
			pt, _ := prev["type"].(string)
			if pt == "user" {
				out[len(out)-1] = MergeUserMessagesMap(prev, m)
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// NormalizeMessagesForAPIGeneric mirrors TS normalizeMessagesForAPI on []map[string]any (see messages_normalize_api_full.go).
func NormalizeMessagesForAPIGeneric(msgs []map[string]any, cfg NormalizeMessagesForAPIConfig) ([]map[string]any, error) {
	return normalizeMessagesForAPIComplete(msgs, cfg)
}

func mergeOrAppendUser(result []map[string]any, u map[string]any) []map[string]any {
	if len(result) == 0 {
		return append(result, u)
	}
	last := result[len(result)-1]
	lt, _ := last["type"].(string)
	if lt == "user" {
		result[len(result)-1] = MergeUserMessagesMap(last, u)
		return result
	}
	return append(result, u)
}

func systemLocalCommandToUserMap(sys map[string]any) map[string]any {
	u := map[string]any{
		"type": "user",
		"message": map[string]any{
			"content": sys["content"],
		},
	}
	if v, ok := sys["uuid"]; ok {
		u["uuid"] = v
	}
	if v, ok := sys["timestamp"]; ok {
		u["timestamp"] = v
	}
	return u
}

func assistantMessageID(m map[string]any) string {
	msg, ok := m["message"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := msg["id"].(string)
	return id
}

func isToolResultUserMessage(m map[string]any) bool {
	t, _ := m["type"].(string)
	if t != "user" {
		return false
	}
	msg, ok := m["message"].(map[string]any)
	if !ok {
		return false
	}
	arr, ok := msg["content"].([]any)
	if !ok {
		return false
	}
	for _, it := range arr {
		b, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := b["type"].(string); typ == "tool_result" {
			return true
		}
	}
	return false
}

func truthy(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func cloneMapJSON(m map[string]any) map[string]any {
	b, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return m
	}
	return out
}

func normalizeUserContentToBlocks(content any) []map[string]any {
	switch v := content.(type) {
	case string:
		return []map[string]any{{"type": "text", "text": v}}
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, it := range v {
			bm, ok := it.(map[string]any)
			if ok {
				out = append(out, bm)
			}
		}
		return out
	default:
		return nil
	}
}

func joinTextAtSeamBlocks(a, b []map[string]any) []map[string]any {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	la := a[len(a)-1]
	fb := b[0]
	lat, _ := la["type"].(string)
	fbt, _ := fb["type"].(string)
	if lat == "text" && fbt == "text" {
		laText, _ := la["text"].(string)
		fbText, _ := fb["text"].(string)
		merged := append(append([]map[string]any{}, a[:len(a)-1]...), map[string]any{
			"type": "text", "text": laText + "\n" + fbText,
		})
		return append(merged, b[1:]...)
	}
	return append(append([]map[string]any{}, a...), b...)
}

func hoistToolResultsBlocks(content []map[string]any) []map[string]any {
	var tr, other []map[string]any
	for _, b := range content {
		typ, _ := b["type"].(string)
		if typ == "tool_result" {
			tr = append(tr, b)
		} else {
			other = append(other, b)
		}
	}
	out := make([]map[string]any, 0, len(content))
	out = append(out, tr...)
	out = append(out, other...)
	return out
}

func blocksToAnySlice(blocks []map[string]any) []any {
	s := make([]any, len(blocks))
	for i := range blocks {
		s[i] = blocks[i]
	}
	return s
}

// GetLastAssistantMessageMap scans from the end for type "assistant".
func GetLastAssistantMessageMap(msgs []map[string]any) (map[string]any, bool) {
	for i := len(msgs) - 1; i >= 0; i-- {
		if t, _ := msgs[i]["type"].(string); t == "assistant" {
			return msgs[i], true
		}
	}
	return nil, false
}

// HasToolCallsInLastAssistantTurnMap mirrors TS hasToolCallsInLastAssistantTurn.
func HasToolCallsInLastAssistantTurnMap(msgs []map[string]any) bool {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if t, _ := m["type"].(string); t != "assistant" {
			continue
		}
		inner, ok := m["message"].(map[string]any)
		if !ok {
			return false
		}
		arr, ok := inner["content"].([]any)
		if !ok {
			return false
		}
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := b["type"].(string); typ == "tool_use" {
				return true
			}
		}
		return false
	}
	return false
}
