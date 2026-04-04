package query

import (
	"bytes"
	"encoding/json"
)

// ToolResultBlock is one tool_result in a user message (Messages API shape).
type ToolResultBlock struct {
	ToolUseID string
	Content   string
}

// AppendAssistantTurnMessage appends an assistant message with optional text and tool_use blocks.
func AppendAssistantTurnMessage(messagesJSON json.RawMessage, text string, uses []ToolUseCall) (json.RawMessage, error) {
	var content []any
	if text != "" {
		content = append(content, map[string]string{"type": "text", "text": text})
	}
	for _, u := range uses {
		var input any
		raw := bytes.TrimSpace(u.Input)
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &input); err != nil {
				return nil, err
			}
		} else {
			input = map[string]any{}
		}
		content = append(content, map[string]any{
			"type":  "tool_use",
			"id":    u.ID,
			"name":  u.Name,
			"input": input,
		})
	}
	if len(content) == 0 {
		content = []any{map[string]string{"type": "text", "text": ""}}
	}
	piece, err := json.Marshal(map[string]any{
		"role":    "assistant",
		"content": content,
	})
	if err != nil {
		return nil, err
	}
	return appendRawMessageToList(messagesJSON, piece)
}

// AppendUserToolResultsMessage appends a user message containing tool_result blocks.
func AppendUserToolResultsMessage(messagesJSON json.RawMessage, results []ToolResultBlock) (json.RawMessage, error) {
	if len(results) == 0 {
		return messagesJSON, nil
	}
	blocks := make([]any, 0, len(results))
	for _, r := range results {
		blocks = append(blocks, map[string]string{
			"type":        "tool_result",
			"tool_use_id": r.ToolUseID,
			"content":     r.Content,
		})
	}
	piece, err := json.Marshal(map[string]any{
		"role":    "user",
		"content": blocks,
	})
	if err != nil {
		return nil, err
	}
	return appendRawMessageToList(messagesJSON, piece)
}

func appendRawMessageToList(messagesJSON json.RawMessage, piece []byte) (json.RawMessage, error) {
	var list []json.RawMessage
	raw := bytes.TrimSpace(messagesJSON)
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, err
		}
	}
	list = append(list, json.RawMessage(piece))
	return json.Marshal(list)
}

// AppendUserTextMessage appends a user message with a single text block to a Messages API JSON array.
func AppendUserTextMessage(messagesJSON json.RawMessage, userText string) (json.RawMessage, error) {
	return appendTextRoleMessage(messagesJSON, "user", userText)
}

// AppendMetaUserTextMessage appends a user nudge for token-budget continuation (query.ts createUserMessage isMeta).
func AppendMetaUserTextMessage(messagesJSON json.RawMessage, text string) (json.RawMessage, error) {
	return AppendUserTextMessage(messagesJSON, text)
}

// AppendAssistantTextMessage appends an assistant message with a single text block.
func AppendAssistantTextMessage(messagesJSON json.RawMessage, assistantText string) (json.RawMessage, error) {
	return appendTextRoleMessage(messagesJSON, "assistant", assistantText)
}

func appendTextRoleMessage(messagesJSON json.RawMessage, role, text string) (json.RawMessage, error) {
	piece, err := json.Marshal(map[string]any{
		"role": role,
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	})
	if err != nil {
		return nil, err
	}
	return appendRawMessageToList(messagesJSON, piece)
}

// InitialUserMessagesJSON builds [{"role":"user","content":[{"type":"text","text":...}]}].
func InitialUserMessagesJSON(userText string) (json.RawMessage, error) {
	return json.Marshal([]map[string]any{
		{
			"role": "user",
			"content": []map[string]string{
				{"type": "text", "text": userText},
			},
		},
	})
}
