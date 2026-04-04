// Pipeline, lookups, filters, stream handling (parity with src/utils/messages.ts).
package messages

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// HookEvent mirrors TS HookEvent string union.
type HookEvent string

// MessageLookups mirrors TS MessageLookups (Go maps replace TS Map/Set).
type MessageLookups struct {
	SiblingToolUseIDs           map[string]map[string]struct{}
	ProgressMessagesByToolUseID map[string][]TSMsg
	InProgressHookCounts        map[string]map[HookEvent]int
	ResolvedHookCounts          map[string]map[HookEvent]int
	ToolResultByToolUseID       map[string]TSMsg
	ToolUseByToolUseID          map[string]map[string]any
	NormalizedMessageCount      int
	ResolvedToolUseIDs          map[string]struct{}
	ErroredToolUseIDs           map[string]struct{}
}

// NewEmptyLookups mirrors EMPTY_LOOKUPS.
func NewEmptyLookups() MessageLookups {
	return MessageLookups{
		SiblingToolUseIDs:           map[string]map[string]struct{}{},
		ProgressMessagesByToolUseID: map[string][]TSMsg{},
		InProgressHookCounts:        map[string]map[HookEvent]int{},
		ResolvedHookCounts:          map[string]map[HookEvent]int{},
		ToolResultByToolUseID:       map[string]TSMsg{},
		ToolUseByToolUseID:          map[string]map[string]any{},
		ResolvedToolUseIDs:          map[string]struct{}{},
		ErroredToolUseIDs:           map[string]struct{}{},
	}
}

func setAdd(m map[string]struct{}, k string) {
	if m == nil {
		return
	}
	m[k] = struct{}{}
}

func cloneTSMsg(m TSMsg) TSMsg {
	b, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var out TSMsg
	_ = json.Unmarshal(b, &out)
	return out
}

// NormalizeMessages mirrors TS normalizeMessages (split multi-block user/assistant turns).
func NormalizeMessages(messages []TSMsg) []TSMsg {
	isNewChain := false
	var out []TSMsg
	for _, message := range messages {
		t, _ := message["type"].(string)
		switch t {
		case "assistant":
			inner, _ := message["message"].(map[string]any)
			content, _ := inner["content"].([]any)
			isNewChain = isNewChain || len(content) > 1
			for idx, block := range content {
				uuid, _ := message["uuid"].(string)
				if isNewChain {
					uuid = DeriveUUID(uuid, idx)
				}
				nm := cloneTSMsg(message)
				nInner, _ := nm["message"].(map[string]any)
				nInner["content"] = []any{block}
				if cm, ok := inner["context_management"]; ok {
					nInner["context_management"] = cm
				} else {
					nInner["context_management"] = nil
				}
				nm["uuid"] = uuid
				out = append(out, nm)
			}
		case "attachment", "progress", "system":
			out = append(out, cloneTSMsg(message))
		case "user":
			inner, _ := message["message"].(map[string]any)
			rawContent := inner["content"]
			if s, ok := rawContent.(string); ok {
				uuid, _ := message["uuid"].(string)
				if isNewChain {
					uuid = DeriveUUID(uuid, 0)
				}
				nm := cloneTSMsg(message)
				nInner, _ := nm["message"].(map[string]any)
				nInner["content"] = []any{map[string]any{"type": "text", "text": s}}
				nm["uuid"] = uuid
				out = append(out, nm)
				continue
			}
			content, _ := rawContent.([]any)
			isNewChain = isNewChain || len(content) > 1
			imageIdx := 0
			for idx, block := range content {
				bm, ok := block.(map[string]any)
				if !ok {
					continue
				}
				opts := CreateUserMessageOpts{
					Content:                   []any{bm},
					IsMeta:                    truthy(message["isMeta"]),
					IsVisibleInTranscriptOnly: truthy(message["isVisibleInTranscriptOnly"]),
					IsVirtual:                 truthy(message["isVirtual"]),
					Timestamp:                 strField(message, "timestamp"),
					Origin:                    originMap(message["origin"]),
				}
				if v, ok := message["toolUseResult"]; ok {
					opts.ToolUseResult = v
				}
				if v, ok := message["mcpMeta"].(map[string]any); ok {
					opts.McpMeta = v
				}
				if bm["type"] == "image" {
					if ids, ok := message["imagePasteIds"].([]any); ok && imageIdx < len(ids) {
						opts.ImagePasteIds = []any{ids[imageIdx]}
					}
					imageIdx++
				}
				uuid, _ := message["uuid"].(string)
				if isNewChain {
					uuid = DeriveUUID(uuid, idx)
				}
				opts.UUID = uuid
				um := CreateUserMessage(opts)
				out = append(out, um)
			}
		default:
			out = append(out, cloneTSMsg(message))
		}
	}
	return out
}

func strField(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

func originMap(v any) map[string]any {
	o, _ := v.(map[string]any)
	return o
}

// IsToolUseRequestMessage mirrors TS isToolUseRequestMessage.
func IsToolUseRequestMessage(msg TSMsg) bool {
	if t, _ := msg["type"].(string); t != "assistant" {
		return false
	}
	inner, ok := msg["message"].(map[string]any)
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

// IsToolUseResultMessage mirrors TS isToolUseResultMessage.
func IsToolUseResultMessage(msg TSMsg) bool {
	if t, _ := msg["type"].(string); t != "user" {
		return false
	}
	if _, ok := msg["toolUseResult"]; ok && msg["toolUseResult"] != nil {
		return true
	}
	inner, ok := msg["message"].(map[string]any)
	if !ok {
		return false
	}
	arr, ok := inner["content"].([]any)
	if !ok || len(arr) == 0 {
		return false
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return false
	}
	ft, _ := first["type"].(string)
	return ft == "tool_result"
}

func isHookAttachmentMessageTS(msg TSMsg) bool {
	if t, _ := msg["type"].(string); t != "attachment" {
		return false
	}
	att, ok := msg["attachment"].(map[string]any)
	if !ok {
		return false
	}
	ty, _ := att["type"].(string)
	switch ty {
	case "hook_blocking_error", "hook_cancelled", "hook_error_during_execution",
		"hook_non_blocking_error", "hook_success", "hook_system_message",
		"hook_additional_context", "hook_stopped_continuation":
		return true
	default:
		return false
	}
}

func hookEventOf(msg TSMsg) (string, bool) {
	att, ok := msg["attachment"].(map[string]any)
	if !ok {
		return "", false
	}
	e, _ := att["hookEvent"].(string)
	return e, e != ""
}

func hookNameOf(msg TSMsg) (string, bool) {
	att, ok := msg["attachment"].(map[string]any)
	if !ok {
		return "", false
	}
	n, _ := att["hookName"].(string)
	return n, n != ""
}

func toolUseIDOfAttachment(msg TSMsg) (string, bool) {
	att, ok := msg["attachment"].(map[string]any)
	if !ok {
		return "", false
	}
	id, _ := att["toolUseID"].(string)
	return id, id != ""
}

type toolUseGroup struct {
	toolUse    TSMsg
	preHooks   []TSMsg
	toolResult TSMsg
	postHooks  []TSMsg
}

// ReorderMessagesInUI mirrors TS reorderMessagesInUI.
func ReorderMessagesInUI(messages []TSMsg, syntheticStreaming []TSMsg) []TSMsg {
	groups := make(map[string]*toolUseGroup)
	for _, message := range messages {
		if IsToolUseRequestMessage(message) {
			inner, _ := message["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			if len(arr) == 0 {
				continue
			}
			first, _ := arr[0].(map[string]any)
			toolUseID, _ := first["id"].(string)
			if toolUseID == "" {
				continue
			}
			if groups[toolUseID] == nil {
				groups[toolUseID] = &toolUseGroup{}
			}
			groups[toolUseID].toolUse = cloneTSMsg(message)
			continue
		}
		if isHookAttachmentMessageTS(message) {
			ev, _ := hookEventOf(message)
			tid, _ := toolUseIDOfAttachment(message)
			if tid == "" {
				continue
			}
			if groups[tid] == nil {
				groups[tid] = &toolUseGroup{}
			}
			if ev == "PreToolUse" {
				groups[tid].preHooks = append(groups[tid].preHooks, cloneTSMsg(message))
			} else if ev == "PostToolUse" {
				groups[tid].postHooks = append(groups[tid].postHooks, cloneTSMsg(message))
			}
			continue
		}
		if t, _ := message["type"].(string); t == "user" {
			inner, _ := message["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			if len(arr) == 0 {
				continue
			}
			first, _ := arr[0].(map[string]any)
			if typ, _ := first["type"].(string); typ != "tool_result" {
				continue
			}
			toolUseID, _ := first["tool_use_id"].(string)
			if toolUseID == "" {
				continue
			}
			if groups[toolUseID] == nil {
				groups[toolUseID] = &toolUseGroup{}
			}
			groups[toolUseID].toolResult = cloneTSMsg(message)
		}
	}

	var result []TSMsg
	processed := make(map[string]struct{})
	for _, message := range messages {
		if IsToolUseRequestMessage(message) {
			inner, _ := message["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			if len(arr) == 0 {
				continue
			}
			first, _ := arr[0].(map[string]any)
			toolUseID, _ := first["id"].(string)
			if toolUseID == "" {
				continue
			}
			if _, done := processed[toolUseID]; done {
				continue
			}
			processed[toolUseID] = struct{}{}
			g := groups[toolUseID]
			if g != nil && g.toolUse != nil {
				result = append(result, g.toolUse)
				result = append(result, g.preHooks...)
				if g.toolResult != nil {
					result = append(result, g.toolResult)
				}
				result = append(result, g.postHooks...)
			}
			continue
		}
		if isHookAttachmentMessageTS(message) {
			ev, _ := hookEventOf(message)
			if ev == "PreToolUse" || ev == "PostToolUse" {
				continue
			}
		}
		if t, _ := message["type"].(string); t == "user" {
			inner, _ := message["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			if len(arr) > 0 {
				first, _ := arr[0].(map[string]any)
				if typ, _ := first["type"].(string); typ == "tool_result" {
					continue
				}
			}
		}
		if t, _ := message["type"].(string); t == "system" {
			st, _ := message["subtype"].(string)
			if st == "api_error" {
				if len(result) > 0 {
					last := result[len(result)-1]
					if lt, _ := last["type"].(string); lt == "system" {
						if lst, _ := last["subtype"].(string); lst == "api_error" {
							result[len(result)-1] = cloneTSMsg(message)
							continue
						}
					}
				}
				result = append(result, cloneTSMsg(message))
				continue
			}
		}
		result = append(result, cloneTSMsg(message))
	}
	for _, m := range syntheticStreaming {
		result = append(result, cloneTSMsg(m))
	}
	if len(result) == 0 {
		return result
	}
	lastIdx := len(result) - 1
	var out []TSMsg
	for i, m := range result {
		if t, _ := m["type"].(string); t == "system" {
			if st, _ := m["subtype"].(string); st == "api_error" {
				if i != lastIdx {
					continue
				}
			}
		}
		out = append(out, m)
	}
	return out
}

func getInProgressHookCountTS(msgs []TSMsg, toolUseID string, hookEvent HookEvent) int {
	n := 0
	for _, msg := range msgs {
		if t, _ := msg["type"].(string); t != "progress" {
			continue
		}
		if msg["parentToolUseID"] != toolUseID {
			continue
		}
		data, _ := msg["data"].(map[string]any)
		if dt, _ := data["type"].(string); dt != "hook_progress" {
			continue
		}
		he, _ := data["hookEvent"].(string)
		if HookEvent(he) == hookEvent {
			n++
		}
	}
	return n
}

func getResolvedHookCountTS(msgs []TSMsg, toolUseID string, hookEvent HookEvent) int {
	names := make(map[string]struct{})
	for _, msg := range msgs {
		if !isHookAttachmentMessageTS(msg) {
			continue
		}
		tid, _ := toolUseIDOfAttachment(msg)
		if tid != toolUseID {
			continue
		}
		ev, _ := hookEventOf(msg)
		if HookEvent(ev) != hookEvent {
			continue
		}
		if hn, ok := hookNameOf(msg); ok {
			names[hn] = struct{}{}
		}
	}
	return len(names)
}

// HasUnresolvedHooks mirrors TS hasUnresolvedHooks.
func HasUnresolvedHooks(messages []TSMsg, toolUseID string, hookEvent HookEvent) bool {
	return getInProgressHookCountTS(messages, toolUseID, hookEvent) > getResolvedHookCountTS(messages, toolUseID, hookEvent)
}

// GetToolResultIDs mirrors TS getToolResultIDs (simplified: first tool_result per normalized user).
func GetToolResultIDs(normalized []TSMsg) map[string]struct{} {
	out := make(map[string]struct{})
	for _, msg := range normalized {
		if t, _ := msg["type"].(string); t != "user" {
			continue
		}
		inner, _ := msg["message"].(map[string]any)
		arr, _ := inner["content"].([]any)
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := b["type"].(string); typ == "tool_result" {
				id, _ := b["tool_use_id"].(string)
				if id != "" {
					out[id] = struct{}{}
				}
			}
		}
	}
	return out
}

// GetSiblingToolUseIDs mirrors TS getSiblingToolUseIDs.
func GetSiblingToolUseIDs(messages []TSMsg, toolUseID string) map[string]struct{} {
	if toolUseID == "" {
		return map[string]struct{}{}
	}
	var unnormalized TSMsg
	for _, m := range messages {
		if t, _ := m["type"].(string); t != "assistant" {
			continue
		}
		inner, _ := m["message"].(map[string]any)
		arr, _ := inner["content"].([]any)
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := b["type"].(string); typ == "tool_use" {
				if id, _ := b["id"].(string); id == toolUseID {
					unnormalized = m
					break
				}
			}
		}
		if unnormalized != nil {
			break
		}
	}
	if unnormalized == nil {
		return map[string]struct{}{}
	}
	inner, _ := unnormalized["message"].(map[string]any)
	msgID, _ := inner["id"].(string)
	sibs := make(map[string]struct{})
	for _, m := range messages {
		if t, _ := m["type"].(string); t != "assistant" {
			continue
		}
		in, _ := m["message"].(map[string]any)
		mid, _ := in["id"].(string)
		if mid != msgID {
			continue
		}
		arr, _ := in["content"].([]any)
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := b["type"].(string); typ == "tool_use" {
				if id, _ := b["id"].(string); id != "" {
					sibs[id] = struct{}{}
				}
			}
		}
	}
	return sibs
}

// BuildMessageLookups mirrors TS buildMessageLookups.
func BuildMessageLookups(normalizedMessages []TSMsg, messages []TSMsg) MessageLookups {
	toolUseIDsByMessageID := make(map[string]map[string]struct{})
	toolUseIDToMessageID := make(map[string]string)
	toolUseByToolUseID := make(map[string]map[string]any)

	for _, msg := range messages {
		if t, _ := msg["type"].(string); t != "assistant" {
			continue
		}
		inner, _ := msg["message"].(map[string]any)
		id, _ := inner["id"].(string)
		if id == "" {
			continue
		}
		if toolUseIDsByMessageID[id] == nil {
			toolUseIDsByMessageID[id] = make(map[string]struct{})
		}
		arr, _ := inner["content"].([]any)
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := b["type"].(string); typ == "tool_use" {
				tuid, _ := b["id"].(string)
				if tuid != "" {
					toolUseIDsByMessageID[id][tuid] = struct{}{}
					toolUseIDToMessageID[tuid] = id
					toolUseByToolUseID[tuid] = b
				}
			}
		}
	}

	siblingToolUseIDs := make(map[string]map[string]struct{})
	for tuid, mid := range toolUseIDToMessageID {
		siblingToolUseIDs[tuid] = toolUseIDsByMessageID[mid]
	}

	lookups := NewEmptyLookups()
	lookups.SiblingToolUseIDs = siblingToolUseIDs
	lookups.ToolUseByToolUseID = toolUseByToolUseID
	lookups.NormalizedMessageCount = len(normalizedMessages)

	resolvedHookNames := make(map[string]map[HookEvent]map[string]struct{})
	resolvedToolUseIDs := lookups.ResolvedToolUseIDs
	erroredToolUseIDs := lookups.ErroredToolUseIDs

	for _, msg := range normalizedMessages {
		if t, _ := msg["type"].(string); t == "progress" {
			tid, _ := msg["parentToolUseID"].(string)
			lookups.ProgressMessagesByToolUseID[tid] = append(lookups.ProgressMessagesByToolUseID[tid], msg)
			data, _ := msg["data"].(map[string]any)
			if dt, _ := data["type"].(string); dt == "hook_progress" {
				he := HookEvent(strField(data, "hookEvent"))
				if lookups.InProgressHookCounts[tid] == nil {
					lookups.InProgressHookCounts[tid] = make(map[HookEvent]int)
				}
				lookups.InProgressHookCounts[tid][he]++
			}
		}
		if t, _ := msg["type"].(string); t == "user" {
			inner, _ := msg["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			for _, it := range arr {
				b, ok := it.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := b["type"].(string); typ == "tool_result" {
					tuid, _ := b["tool_use_id"].(string)
					if tuid != "" {
						lookups.ToolResultByToolUseID[tuid] = msg
						setAdd(resolvedToolUseIDs, tuid)
						if truthy(b["is_error"]) {
							setAdd(erroredToolUseIDs, tuid)
						}
					}
				}
			}
		}
		if t, _ := msg["type"].(string); t == "assistant" {
			inner, _ := msg["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			for _, it := range arr {
				b, ok := it.(map[string]any)
				if !ok {
					continue
				}
				if _, has := b["tool_use_id"]; has {
					if s, ok := b["tool_use_id"].(string); ok && s != "" {
						setAdd(resolvedToolUseIDs, s)
					}
				}
				bt, _ := b["type"].(string)
				if bt == "advisor_tool_result" {
					tuid, _ := b["tool_use_id"].(string)
					innerC, _ := b["content"].(map[string]any)
					if innerC != nil {
						if ct, _ := innerC["type"].(string); ct == "advisor_tool_result_error" && tuid != "" {
							setAdd(erroredToolUseIDs, tuid)
						}
					}
				}
			}
		}
		if isHookAttachmentMessageTS(msg) {
			tid, _ := toolUseIDOfAttachment(msg)
			ev := HookEvent(strField(msg["attachment"].(map[string]any), "hookEvent"))
			hn, ok := hookNameOf(msg)
			if ok && tid != "" {
				if resolvedHookNames[tid] == nil {
					resolvedHookNames[tid] = make(map[HookEvent]map[string]struct{})
				}
				if resolvedHookNames[tid][ev] == nil {
					resolvedHookNames[tid][ev] = make(map[string]struct{})
				}
				resolvedHookNames[tid][ev][hn] = struct{}{}
			}
		}
	}

	for tid, byEv := range resolvedHookNames {
		cm := make(map[HookEvent]int)
		for ev, names := range byEv {
			cm[ev] = len(names)
		}
		lookups.ResolvedHookCounts[tid] = cm
	}

	var lastAssistantMsgID string
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if t, _ := last["type"].(string); t == "assistant" {
			inner, _ := last["message"].(map[string]any)
			lastAssistantMsgID, _ = inner["id"].(string)
		}
	}
	for _, msg := range normalizedMessages {
		if t, _ := msg["type"].(string); t != "assistant" {
			continue
		}
		inner, _ := msg["message"].(map[string]any)
		mid, _ := inner["id"].(string)
		if mid == lastAssistantMsgID {
			continue
		}
		arr, _ := inner["content"].([]any)
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			bt, _ := b["type"].(string)
			if bt != "server_tool_use" && bt != "mcp_tool_use" {
				continue
			}
			id, _ := b["id"].(string)
			if id == "" {
				continue
			}
			if _, ok := resolvedToolUseIDs[id]; !ok {
				setAdd(resolvedToolUseIDs, id)
				setAdd(erroredToolUseIDs, id)
			}
		}
	}
	return lookups
}

// BuildSubagentLookups mirrors TS buildSubagentLookups.
func BuildSubagentLookups(messages []struct{ Message TSMsg }) (MessageLookups, map[string]struct{}) {
	lk := NewEmptyLookups()
	inProgress := make(map[string]struct{})
	for _, wrap := range messages {
		msg := wrap.Message
		if t, _ := msg["type"].(string); t == "assistant" {
			inner, _ := msg["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			for _, it := range arr {
				b, ok := it.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := b["type"].(string); typ == "tool_use" {
					id, _ := b["id"].(string)
					if id != "" {
						lk.ToolUseByToolUseID[id] = b
					}
				}
			}
		} else if t, _ := msg["type"].(string); t == "user" {
			inner, _ := msg["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			for _, it := range arr {
				b, ok := it.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := b["type"].(string); typ == "tool_result" {
					id, _ := b["tool_use_id"].(string)
					if id != "" {
						setAdd(lk.ResolvedToolUseIDs, id)
						lk.ToolResultByToolUseID[id] = msg
					}
				}
			}
		}
	}
	for id := range lk.ToolUseByToolUseID {
		if _, ok := lk.ResolvedToolUseIDs[id]; !ok {
			inProgress[id] = struct{}{}
		}
	}
	return lk, inProgress
}

// GetSiblingToolUseIDsFromLookup mirrors TS getSiblingToolUseIDsFromLookup.
func GetSiblingToolUseIDsFromLookup(message TSMsg, lookups MessageLookups) map[string]struct{} {
	tid := GetToolUseID(message)
	if tid == nil {
		return map[string]struct{}{}
	}
	if s, ok := lookups.SiblingToolUseIDs[*tid]; ok && s != nil {
		return s
	}
	return map[string]struct{}{}
}

// GetProgressMessagesFromLookup mirrors TS getProgressMessagesFromLookup.
func GetProgressMessagesFromLookup(message TSMsg, lookups MessageLookups) []TSMsg {
	tid := GetToolUseID(message)
	if tid == nil {
		return nil
	}
	return lookups.ProgressMessagesByToolUseID[*tid]
}

// HasUnresolvedHooksFromLookup mirrors TS hasUnresolvedHooksFromLookup.
func HasUnresolvedHooksFromLookup(toolUseID string, hookEvent HookEvent, lookups MessageLookups) bool {
	inProg := 0
	if m, ok := lookups.InProgressHookCounts[toolUseID]; ok {
		inProg = m[hookEvent]
	}
	res := 0
	if m, ok := lookups.ResolvedHookCounts[toolUseID]; ok {
		res = m[hookEvent]
	}
	return inProg > res
}

// GetToolUseIDs mirrors TS getToolUseIDs.
func GetToolUseIDs(normalized []TSMsg) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range normalized {
		if t, _ := m["type"].(string); t != "assistant" {
			continue
		}
		inner, _ := m["message"].(map[string]any)
		arr, _ := inner["content"].([]any)
		if len(arr) == 0 {
			continue
		}
		first, _ := arr[0].(map[string]any)
		if typ, _ := first["type"].(string); typ != "tool_use" {
			continue
		}
		id, _ := first["id"].(string)
		if id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}

// GetToolUseID mirrors TS getToolUseID.
func GetToolUseID(message TSMsg) *string {
	t, _ := message["type"].(string)
	switch t {
	case "attachment":
		if isHookAttachmentMessageTS(message) {
			if id, ok := toolUseIDOfAttachment(message); ok {
				return &id
			}
		}
		return nil
	case "assistant":
		inner, _ := message["message"].(map[string]any)
		arr, _ := inner["content"].([]any)
		if len(arr) == 0 {
			return nil
		}
		first, _ := arr[0].(map[string]any)
		if typ, _ := first["type"].(string); typ != "tool_use" {
			return nil
		}
		id, _ := first["id"].(string)
		if id == "" {
			return nil
		}
		return &id
	case "user":
		if s, ok := message["sourceToolUseID"].(string); ok && s != "" {
			return &s
		}
		inner, _ := message["message"].(map[string]any)
		arr, _ := inner["content"].([]any)
		if len(arr) == 0 {
			return nil
		}
		first, _ := arr[0].(map[string]any)
		if typ, _ := first["type"].(string); typ != "tool_result" {
			return nil
		}
		id, _ := first["tool_use_id"].(string)
		if id == "" {
			return nil
		}
		return &id
	case "progress":
		id, _ := message["toolUseID"].(string)
		if id == "" {
			return nil
		}
		return &id
	case "system":
		st, _ := message["subtype"].(string)
		if st == "informational" {
			if id, ok := message["toolUseID"].(string); ok && id != "" {
				return &id
			}
		}
		return nil
	default:
		return nil
	}
}

// ReorderAttachmentsForAPI is TS reorderAttachmentsForAPI on TSMsg slices.
func ReorderAttachmentsForAPI(messages []TSMsg) []TSMsg {
	in := make([]map[string]any, len(messages))
	for i := range messages {
		in[i] = map[string]any(messages[i])
	}
	out := ReorderAttachmentsForAPIGeneric(in)
	res := make([]TSMsg, len(out))
	for i := range out {
		res[i] = TSMsg(out[i])
	}
	return res
}

// StripToolReferenceBlocksFromUserMessage TS API on TSMsg.
func StripToolReferenceBlocksFromUserMessage(msg TSMsg) TSMsg {
	return TSMsg(StripToolReferenceBlocksFromUserMessageMap(map[string]any(msg)))
}

// StripCallerFieldFromAssistantMessage TS API on TSMsg.
func StripCallerFieldFromAssistantMessage(msg TSMsg) TSMsg {
	return TSMsg(StripCallerFieldFromAssistantMessageMap(map[string]any(msg)))
}

// MergeUserMessages is TS mergeUserMessages on TSMsg.
func MergeUserMessages(a, b TSMsg) TSMsg {
	return TSMsg(MergeUserMessagesMap(map[string]any(a), map[string]any(b)))
}

// MergeAssistantMessages is TS mergeAssistantMessages.
func MergeAssistantMessages(a, b TSMsg) TSMsg {
	return TSMsg(MergeAssistantMessagesMap(map[string]any(a), map[string]any(b)))
}

// MergeUserMessagesAndToolResults is TS mergeUserMessagesAndToolResults.
func MergeUserMessagesAndToolResults(a, b TSMsg) TSMsg {
	return TSMsg(MergeUserMessagesAndToolResultsMap(map[string]any(a), map[string]any(b)))
}

// NormalizeToolInputFunc optionally normalizes tool_use input (TS normalizeToolInput).
type NormalizeToolInputFunc func(toolName string, input map[string]any) map[string]any

// NormalizeContentFromAPI mirrors TS normalizeContentFromAPI (tools slice = tool names for matching).
func NormalizeContentFromAPI(contentBlocks []any, toolNames map[string]struct{}, normalizeInput NormalizeToolInputFunc) []any {
	if len(contentBlocks) == 0 {
		return contentBlocks
	}
	out := make([]any, 0, len(contentBlocks))
	for _, raw := range contentBlocks {
		block, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		typ, _ := block["type"].(string)
		switch typ {
		case "tool_use":
			inp := block["input"]
			var normalized any
			if s, ok := inp.(string); ok {
				var parsed any
				_ = json.Unmarshal([]byte(s), &parsed)
				if parsed == nil && s != "" {
					parsed = map[string]any{}
				}
				if parsed == nil {
					parsed = map[string]any{}
				}
				normalized = parsed
			} else {
				normalized = inp
			}
			if m, ok := normalized.(map[string]any); ok && normalizeInput != nil {
				name, _ := block["name"].(string)
				if _, known := toolNames[name]; known {
					normalized = normalizeInput(name, m)
				}
			}
			nb := cloneMapJSON(block)
			nb["input"] = normalized
			out = append(out, nb)
		case "text":
			out = append(out, block)
		case "code_execution_tool_result", "mcp_tool_use", "mcp_tool_result", "container_upload":
			out = append(out, block)
		case "server_tool_use":
			nb := cloneMapJSON(block)
			if s, ok := nb["input"].(string); ok {
				var parsed any
				_ = json.Unmarshal([]byte(s), &parsed)
				if parsed == nil {
					parsed = map[string]any{}
				}
				nb["input"] = parsed
			}
			out = append(out, nb)
		default:
			out = append(out, block)
		}
	}
	return out
}

var stripPromptXMLTagsRes = []*regexp.Regexp{
	regexp.MustCompile(`(?s)<commit_analysis>.*?</commit_analysis>\n?`),
	regexp.MustCompile(`(?s)<context>.*?</context>\n?`),
	regexp.MustCompile(`(?s)<function_analysis>.*?</function_analysis>\n?`),
	regexp.MustCompile(`(?s)<pr_analysis>.*?</pr_analysis>\n?`),
}

// StripPromptXMLTags mirrors TS stripPromptXMLTags.
func StripPromptXMLTags(content string) string {
	s := content
	for _, re := range stripPromptXMLTagsRes {
		s = re.ReplaceAllString(s, "")
	}
	return strings.TrimSpace(s)
}

// IsEmptyMessageText mirrors TS isEmptyMessageText.
func IsEmptyMessageText(text string) bool {
	t := strings.TrimSpace(StripPromptXMLTags(text))
	return t == "" || strings.TrimSpace(text) == NOContentMessage
}

// ExtractTextContent mirrors TS extractTextContent.
func ExtractTextContent(blocks []any, separator string) string {
	var parts []string
	for _, it := range blocks {
		b, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := b["type"].(string); typ == "text" {
			parts = append(parts, strField(b, "text"))
		}
	}
	return strings.Join(parts, separator)
}

// GetContentText mirrors TS getContentText.
func GetContentText(content any) *string {
	switch v := content.(type) {
	case string:
		return &v
	case []any:
		s := strings.TrimSpace(ExtractTextContent(v, "\n"))
		if s == "" {
			return nil
		}
		return &s
	default:
		return nil
	}
}

// GetAssistantMessageText mirrors TS getAssistantMessageText.
func GetAssistantMessageText(message TSMsg) *string {
	if t, _ := message["type"].(string); t != "assistant" {
		return nil
	}
	inner, _ := message["message"].(map[string]any)
	arr, ok := inner["content"].([]any)
	if !ok {
		return nil
	}
	s := strings.TrimSpace(ExtractTextContent(arr, "\n"))
	if s == "" {
		return nil
	}
	return &s
}

// GetUserMessageText mirrors TS getUserMessageText.
func GetUserMessageText(message TSMsg) *string {
	if t, _ := message["type"].(string); t != "user" {
		return nil
	}
	inner, _ := message["message"].(map[string]any)
	return GetContentText(inner["content"])
}

// TextForResubmit mirrors TS textForResubmit.
func TextForResubmit(msg TSMsg) (text string, mode string, ok bool) {
	content := GetUserMessageText(msg)
	if content == nil {
		return "", "", false
	}
	if bash := ExtractTag(*content, BashInputTag); bash != nil {
		return *bash, "bash", true
	}
	if cmd := ExtractTag(*content, CommandNameTag); cmd != nil {
		args := ExtractTag(*content, CommandArgsTag)
		a := ""
		if args != nil {
			a = *args
		}
		return strings.TrimSpace(*cmd + " " + a), "prompt", true
	}
	return StripIdeContextTags(*content), "prompt", true
}

// StripIdeContextTags strips common IDE context wrappers (minimal parity).
func StripIdeContextTags(s string) string {
	return s
}

// FilterUnresolvedToolUses mirrors TS filterUnresolvedToolUses.
func FilterUnresolvedToolUses(messages []TSMsg) []TSMsg {
	toolUseIds := make(map[string]struct{})
	toolResultIds := make(map[string]struct{})
	for _, msg := range messages {
		t, _ := msg["type"].(string)
		if t != "user" && t != "assistant" {
			continue
		}
		inner, _ := msg["message"].(map[string]any)
		arr, ok := inner["content"].([]any)
		if !ok {
			continue
		}
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := b["type"].(string)
			if typ == "tool_use" {
				if id, _ := b["id"].(string); id != "" {
					toolUseIds[id] = struct{}{}
				}
			}
			if typ == "tool_result" {
				if id, _ := b["tool_use_id"].(string); id != "" {
					toolResultIds[id] = struct{}{}
				}
			}
		}
	}
	unresolved := make(map[string]struct{})
	for id := range toolUseIds {
		if _, ok := toolResultIds[id]; !ok {
			unresolved[id] = struct{}{}
		}
	}
	if len(unresolved) == 0 {
		return messages
	}
	var out []TSMsg
	for _, msg := range messages {
		if t, _ := msg["type"].(string); t != "assistant" {
			out = append(out, msg)
			continue
		}
		inner, _ := msg["message"].(map[string]any)
		arr, ok := inner["content"].([]any)
		if !ok {
			out = append(out, msg)
			continue
		}
		var ids []string
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := b["type"].(string); typ == "tool_use" {
				if id, _ := b["id"].(string); id != "" {
					ids = append(ids, id)
				}
			}
		}
		if len(ids) == 0 {
			out = append(out, msg)
			continue
		}
		allUnres := true
		for _, id := range ids {
			if _, ok := unresolved[id]; !ok {
				allUnres = false
				break
			}
		}
		if !allUnres {
			out = append(out, msg)
		}
	}
	return out
}

// StreamingToolUse mirrors TS type.
type StreamingToolUse struct {
	Index             int
	ContentBlock      map[string]any
	UnparsedToolInput string
}

// StreamingThinking mirrors TS type.
type StreamingThinking struct {
	Thinking         string
	IsStreaming      bool
	StreamingEndedAt *int64
}

// SpinnerMode mirrors TS SpinnerMode.
type SpinnerMode string

// HandleMessageFromStream mirrors TS handleMessageFromStream using map-shaped events.
func HandleMessageFromStream(
	message TSMsg,
	onMessage func(TSMsg),
	onUpdateLength func(string),
	onSetStreamMode func(SpinnerMode),
	onStreamingToolUses func(func([]StreamingToolUse) []StreamingToolUse),
	onTombstone func(TSMsg),
	onStreamingThinking func(func(*StreamingThinking) *StreamingThinking),
	onApiMetrics func(map[string]any),
	onStreamingText func(func(*string) *string),
) {
	t, _ := message["type"].(string)
	if t != "stream_event" && t != "stream_request_start" {
		if t == "tombstone" {
			if inner, ok := message["message"].(map[string]any); ok && onTombstone != nil {
				onTombstone(TSMsg(inner))
			}
			return
		}
		if t == "tool_use_summary" {
			return
		}
		if t == "assistant" && onStreamingThinking != nil {
			inner, _ := message["message"].(map[string]any)
			arr, _ := inner["content"].([]any)
			for _, it := range arr {
				b, ok := it.(map[string]any)
				if !ok {
					continue
				}
				if typ, _ := b["type"].(string); typ == "thinking" {
					th, _ := b["thinking"].(string)
					now := timeNowMs()
					onStreamingThinking(func(_ *StreamingThinking) *StreamingThinking {
						return &StreamingThinking{Thinking: th, IsStreaming: false, StreamingEndedAt: &now}
					})
					break
				}
			}
		}
		if onStreamingText != nil {
			onStreamingText(func(_ *string) *string { return nil })
		}
		if onMessage != nil {
			onMessage(message)
		}
		return
	}
	if t == "stream_request_start" {
		if onSetStreamMode != nil {
			onSetStreamMode("requesting")
		}
		return
	}
	ev, _ := message["event"].(map[string]any)
	evType, _ := ev["type"].(string)
	if evType == "message_start" {
		if ttft, ok := message["ttftMs"].(float64); ok && onApiMetrics != nil {
			onApiMetrics(map[string]any{"ttftMs": int(ttft)})
		}
	}
	if evType == "message_stop" {
		if onSetStreamMode != nil {
			onSetStreamMode("tool-use")
		}
		if onStreamingToolUses != nil {
			onStreamingToolUses(func(_ []StreamingToolUse) []StreamingToolUse { return nil })
		}
		return
	}
	switch evType {
	case "content_block_start":
		if onStreamingText != nil {
			onStreamingText(func(_ *string) *string { return nil })
		}
		cb, _ := ev["content_block"].(map[string]any)
		cbt, _ := cb["type"].(string)
		switch cbt {
		case "thinking", "redacted_thinking":
			if onSetStreamMode != nil {
				onSetStreamMode("thinking")
			}
		case "text":
			if onSetStreamMode != nil {
				onSetStreamMode("responding")
			}
		case "tool_use":
			if onSetStreamMode != nil {
				onSetStreamMode("tool-input")
			}
			idx, _ := ev["index"].(float64)
			if onStreamingToolUses != nil {
				onStreamingToolUses(func(cur []StreamingToolUse) []StreamingToolUse {
					return append(cur, StreamingToolUse{
						Index:             int(idx),
						ContentBlock:      cb,
						UnparsedToolInput: "",
					})
				})
			}
		default:
			if onSetStreamMode != nil {
				onSetStreamMode("tool-input")
			}
		}
	case "content_block_delta":
		delta, _ := ev["delta"].(map[string]any)
		dt, _ := delta["type"].(string)
		switch dt {
		case "text_delta":
			tx, _ := delta["text"].(string)
			if onUpdateLength != nil {
				onUpdateLength(tx)
			}
			if onStreamingText != nil {
				onStreamingText(func(cur *string) *string {
					prev := ""
					if cur != nil {
						prev = *cur
					}
					s := prev + tx
					return &s
				})
			}
		case "input_json_delta":
			part, _ := delta["partial_json"].(string)
			if onUpdateLength != nil {
				onUpdateLength(part)
			}
			idx, _ := ev["index"].(float64)
			if onStreamingToolUses != nil {
				onStreamingToolUses(func(cur []StreamingToolUse) []StreamingToolUse {
					var out []StreamingToolUse
					for _, el := range cur {
						if el.Index == int(idx) {
							el.UnparsedToolInput += part
						}
						out = append(out, el)
					}
					return out
				})
			}
		case "thinking_delta":
			if onUpdateLength != nil {
				th, _ := delta["thinking"].(string)
				onUpdateLength(th)
			}
		}
	case "message_delta":
		if onSetStreamMode != nil {
			onSetStreamMode("responding")
		}
	default:
		if onSetStreamMode != nil {
			onSetStreamMode("responding")
		}
	}
}

func timeNowMs() int64 { return time.Now().UnixMilli() }
