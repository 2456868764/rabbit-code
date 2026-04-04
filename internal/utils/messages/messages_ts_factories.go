// TS-shaped message factories and string helpers (parity with src/utils/messages.ts).
package messages

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TSMsg is a loose JSON object matching Claude Code Message unions (same role as TS Message).
type TSMsg map[string]any

// NOContentMessage mirrors constants/messages.ts NO_CONTENT_MESSAGE.
const NOContentMessage = "(no content)"

// XML tag names (constants/xml.ts).
const (
	CommandNameTag        = "command-name"
	CommandMessageTag     = "command-message"
	CommandArgsTag        = "command-args"
	BashInputTag          = "bash-input"
	LocalCommandStdoutTag = "local-command-stdout"
	LocalCommandCaveatTag = "local-command-caveat"
)

// Tool name constants used in attachment / prompt strings (src/tools/*/prompt|constants).
const (
	ToolNameRead            = "Read"
	ToolNameWrite           = "Write"
	ToolNameEdit            = "Edit"
	ToolNameBash            = "Bash"
	ToolNameGlob            = "Glob"
	ToolNameGrep            = "Grep"
	ToolNameAskUserQuestion = "AskUserQuestion"
	ToolNameExitPlanModeV2  = "ExitPlanMode"
	ToolNameAgent           = "Agent"
	ToolNameTaskLegacy      = "Task"
	ToolNameTaskCreate      = "TaskCreate"
	ToolNameTaskUpdate      = "TaskUpdate"
	ToolNameTaskOutput      = "TaskOutput"
	ToolNameSendMessage     = "SendMessage"
	ToolNameSkill           = "Skill"
	ToolNameTodoWrite       = "TodoWrite"
	MaxLinesToRead          = 2000
)

// CompanionIntroText mirrors buddy/prompt.ts companionIntroText.
func CompanionIntroText(name, species string) string {
	return `# Companion

A small ` + species + ` named ` + name + ` sits beside the user's input box and occasionally comments in a speech bubble. You're not ` + name + ` — it's a separate watcher.

When the user addresses ` + name + ` directly (by name), its bubble will answer. Your job in that moment is to stay out of the way: respond in ONE line or less, or just answer any part of the message meant for you. Don't explain that you're not ` + name + ` — they know. Don't narrate what ` + name + ` might say — the bubble handles that.`
}

func tsJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func tsRandomUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		b[10:16],
	)
}

func tsISO8601() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func tsDefaultUsage() map[string]any {
	return map[string]any{
		"input_tokens":                0,
		"output_tokens":               0,
		"cache_creation_input_tokens": 0,
		"cache_read_input_tokens":     0,
		"server_tool_use":             map[string]any{"web_search_requests": 0, "web_fetch_requests": 0},
		"service_tier":                nil,
		"cache_creation":              map[string]any{"ephemeral_1h_input_tokens": 0, "ephemeral_5m_input_tokens": 0},
		"inference_geo":               nil,
		"iterations":                  nil,
		"speed":                       nil,
	}
}

// DeriveUUID mirrors TS deriveUUID (stable UUID-shaped suffix from parent + index).
func DeriveUUID(parentUUID string, index int) string {
	if len(parentUUID) < 24 {
		parentUUID = parentUUID + strings.Repeat("0", 24-len(parentUUID))
	}
	hex := fmt.Sprintf("%012x", index)
	if len(hex) > 12 {
		hex = hex[len(hex)-12:]
	}
	return parentUUID[:24] + hex
}

// CreateAssistantMessageOpts mirrors createAssistantMessage parameters.
type CreateAssistantMessageOpts struct {
	Content   any // string | []content block
	Usage     map[string]any
	IsVirtual bool
}

// CreateAssistantMessage mirrors TS createAssistantMessage.
func CreateAssistantMessage(opts CreateAssistantMessageOpts) TSMsg {
	content := opts.Content
	var blocks []any
	switch c := content.(type) {
	case string:
		text := c
		if text == "" {
			text = NOContentMessage
		}
		blocks = []any{map[string]any{"type": "text", "text": text}}
	case []any:
		blocks = c
	case nil:
		blocks = []any{map[string]any{"type": "text", "text": NOContentMessage}}
	default:
		blocks = []any{map[string]any{"type": "text", "text": NOContentMessage}}
	}
	usage := opts.Usage
	if usage == nil {
		usage = tsDefaultUsage()
	}
	msg := map[string]any{
		"id":                 tsRandomUUID(),
		"container":          nil,
		"model":              SyntheticModel,
		"role":               "assistant",
		"stop_reason":        "stop_sequence",
		"stop_sequence":      "",
		"type":               "message",
		"usage":              usage,
		"content":            blocks,
		"context_management": nil,
	}
	m := TSMsg{
		"type":      "assistant",
		"uuid":      tsRandomUUID(),
		"timestamp": tsISO8601(),
		"message":   msg,
	}
	if opts.IsVirtual {
		m["isVirtual"] = true
	}
	return m
}

// CreateAssistantAPIErrorMessageOpts mirrors createAssistantAPIErrorMessage.
type CreateAssistantAPIErrorMessageOpts struct {
	Content      string
	APIError     any
	Error        any
	ErrorDetails string
}

// CreateAssistantAPIErrorMessage mirrors TS createAssistantAPIErrorMessage.
func CreateAssistantAPIErrorMessage(opts CreateAssistantAPIErrorMessageOpts) TSMsg {
	text := opts.Content
	if text == "" {
		text = NOContentMessage
	}
	m := CreateAssistantMessage(CreateAssistantMessageOpts{
		Content: []any{map[string]any{"type": "text", "text": text}},
	})
	m["isApiErrorMessage"] = true
	if opts.APIError != nil {
		m["apiError"] = opts.APIError
	}
	if opts.Error != nil {
		m["error"] = opts.Error
	}
	if opts.ErrorDetails != "" {
		m["errorDetails"] = opts.ErrorDetails
	}
	return m
}

// CreateUserMessageOpts mirrors createUserMessage optional fields.
type CreateUserMessageOpts struct {
	Content                   any
	IsMeta                    bool
	IsVisibleInTranscriptOnly bool
	IsVirtual                 bool
	IsCompactSummary          bool
	SummarizeMetadata         map[string]any
	ToolUseResult             any
	McpMeta                   map[string]any
	UUID                      string
	Timestamp                 string
	ImagePasteIds             []any
	SourceToolAssistantUUID   string
	PermissionMode            string
	Origin                    map[string]any
}

// CreateUserMessage mirrors TS createUserMessage.
func CreateUserMessage(opts CreateUserMessageOpts) TSMsg {
	content := opts.Content
	if content == nil {
		content = NOContentMessage
	}
	m := TSMsg{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": content,
		},
	}
	if opts.UUID != "" {
		m["uuid"] = opts.UUID
	} else {
		m["uuid"] = tsRandomUUID()
	}
	if opts.Timestamp != "" {
		m["timestamp"] = opts.Timestamp
	} else {
		m["timestamp"] = tsISO8601()
	}
	if opts.IsMeta {
		m["isMeta"] = true
	}
	if opts.IsVisibleInTranscriptOnly {
		m["isVisibleInTranscriptOnly"] = true
	}
	if opts.IsVirtual {
		m["isVirtual"] = true
	}
	if opts.IsCompactSummary {
		m["isCompactSummary"] = true
	}
	if opts.SummarizeMetadata != nil {
		m["summarizeMetadata"] = opts.SummarizeMetadata
	}
	if opts.ToolUseResult != nil {
		m["toolUseResult"] = opts.ToolUseResult
	}
	if opts.McpMeta != nil {
		m["mcpMeta"] = opts.McpMeta
	}
	if len(opts.ImagePasteIds) > 0 {
		m["imagePasteIds"] = opts.ImagePasteIds
	}
	if opts.SourceToolAssistantUUID != "" {
		m["sourceToolAssistantUUID"] = opts.SourceToolAssistantUUID
	}
	if opts.PermissionMode != "" {
		m["permissionMode"] = opts.PermissionMode
	}
	if opts.Origin != nil {
		m["origin"] = opts.Origin
	}
	return m
}

// PrepareUserContent mirrors TS prepareUserContent.
func PrepareUserContent(inputString string, preceding []any) any {
	if len(preceding) == 0 {
		return inputString
	}
	out := append(append([]any{}, preceding...), map[string]any{"type": "text", "text": inputString})
	return out
}

// CreateUserInterruptionMessage mirrors TS createUserInterruptionMessage.
func CreateUserInterruptionMessage(toolUse bool) TSMsg {
	text := InterruptMessage
	if toolUse {
		text = InterruptMessageForToolUse
	}
	return CreateUserMessage(CreateUserMessageOpts{
		Content: []any{map[string]any{"type": "text", "text": text}},
	})
}

// CreateSyntheticUserCaveatMessage mirrors TS createSyntheticUserCaveatMessage.
func CreateSyntheticUserCaveatMessage() TSMsg {
	return CreateUserMessage(CreateUserMessageOpts{
		Content: fmt.Sprintf(
			"<%s>Caveat: The messages below were generated by the user while running local commands. DO NOT respond to these messages or otherwise consider them in your response unless the user explicitly asks you to.</%s>",
			LocalCommandCaveatTag, LocalCommandCaveatTag,
		),
		IsMeta: true,
	})
}

// FormatCommandInputTags mirrors TS formatCommandInputTags.
func FormatCommandInputTags(commandName, args string) string {
	return fmt.Sprintf("<%s>/%s</%s>\n            <%s>%s</%s>\n            <%s>%s</%s>",
		CommandNameTag, commandName, CommandNameTag,
		CommandMessageTag, commandName, CommandMessageTag,
		CommandArgsTag, args, CommandArgsTag,
	)
}

// CreateModelSwitchBreadcrumbs mirrors TS createModelSwitchBreadcrumbs.
func CreateModelSwitchBreadcrumbs(modelArg, resolvedDisplay string) []TSMsg {
	return []TSMsg{
		CreateSyntheticUserCaveatMessage(),
		CreateUserMessage(CreateUserMessageOpts{Content: FormatCommandInputTags("model", modelArg)}),
		CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf("<%s>Set model to %s</%s>", LocalCommandStdoutTag, resolvedDisplay, LocalCommandStdoutTag),
		}),
	}
}

// CreateProgressMessage mirrors TS createProgressMessage (data is any Progress shape).
func CreateProgressMessage(toolUseID, parentToolUseID string, data any) TSMsg {
	return TSMsg{
		"type":            "progress",
		"data":            data,
		"toolUseID":       toolUseID,
		"parentToolUseID": parentToolUseID,
		"uuid":            tsRandomUUID(),
		"timestamp":       tsISO8601(),
	}
}

// CreateToolResultStopMessage mirrors TS createToolResultStopMessage.
func CreateToolResultStopMessage(toolUseID string) map[string]any {
	return map[string]any{
		"type":        "tool_result",
		"content":     CancelMessage,
		"is_error":    true,
		"tool_use_id": toolUseID,
	}
}

// ExtractTag mirrors TS extractTag (depth-balanced first match).
func ExtractTag(html, tagName string) *string {
	html = strings.TrimSpace(html)
	tagName = strings.TrimSpace(tagName)
	if html == "" || tagName == "" {
		return nil
	}
	escaped := regexp.QuoteMeta(tagName)
	openRe := regexp.MustCompile(`(?i)<` + escaped + `(?:\s+[^>]*)?>`)
	closeRe := regexp.MustCompile(`(?i)</` + escaped + `>`)
	pattern := regexp.MustCompile(`(?i)<` + escaped + `(?:\s+[^>]*)?>([\s\S]*?)</` + escaped + `>`)

	lastIdx := 0
	for lastIdx < len(html) {
		loc := pattern.FindStringSubmatchIndex(html[lastIdx:])
		if loc == nil {
			return nil
		}
		matchStart := lastIdx + loc[0]
		innerStart := lastIdx + loc[2]
		innerEnd := lastIdx + loc[3]
		matchEnd := lastIdx + loc[1]
		content := html[innerStart:innerEnd]
		beforeMatch := html[lastIdx:matchStart]
		depth := 0
		tmp := beforeMatch
		for {
			idx := openRe.FindStringIndex(tmp)
			if idx == nil {
				break
			}
			depth++
			tmp = tmp[idx[1]:]
		}
		tmp = beforeMatch
		for {
			idx := closeRe.FindStringIndex(tmp)
			if idx == nil {
				break
			}
			depth--
			tmp = tmp[idx[1]:]
		}
		if depth == 0 && content != "" {
			return &content
		}
		lastIdx = matchEnd
	}
	return nil
}

// IsNotEmptyMessage mirrors TS isNotEmptyMessage.
func IsNotEmptyMessage(msg TSMsg) bool {
	t, _ := msg["type"].(string)
	if t == "progress" || t == "attachment" || t == "system" {
		return true
	}
	inner, ok := msg["message"].(map[string]any)
	if !ok {
		return false
	}
	content := inner["content"]
	if s, ok := content.(string); ok {
		return strings.TrimSpace(s) != ""
	}
	arr, ok := content.([]any)
	if !ok || len(arr) == 0 {
		return false
	}
	if len(arr) > 1 {
		return true
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return true
	}
	ft, _ := first["type"].(string)
	if ft != "text" {
		return true
	}
	tx, _ := first["text"].(string)
	tx = strings.TrimSpace(tx)
	return tx != "" && tx != NOContentMessage && tx != InterruptMessageForToolUse
}

// IsSyntheticMessage mirrors TS isSyntheticMessage (same rules as IsSyntheticMessageMap).
func IsSyntheticMessage(msg TSMsg) bool { return IsSyntheticMessageMap(msg) }

// SyntheticMessageSet returns texts in TS SYNTHETIC_MESSAGES.
func SyntheticMessageSet() map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range SyntheticMessageTexts() {
		m[s] = struct{}{}
	}
	return m
}
