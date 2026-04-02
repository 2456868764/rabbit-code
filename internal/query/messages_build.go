package query

import (
	"bytes"
	"encoding/json"
)

// AppendUserTextMessage appends a user message with a single text block to a Messages API JSON array.
func AppendUserTextMessage(messagesJSON json.RawMessage, userText string) (json.RawMessage, error) {
	return appendTextRoleMessage(messagesJSON, "user", userText)
}

// AppendAssistantTextMessage appends an assistant message with a single text block.
func AppendAssistantTextMessage(messagesJSON json.RawMessage, assistantText string) (json.RawMessage, error) {
	return appendTextRoleMessage(messagesJSON, "assistant", assistantText)
}

func appendTextRoleMessage(messagesJSON json.RawMessage, role, text string) (json.RawMessage, error) {
	var list []json.RawMessage
	raw := bytes.TrimSpace(messagesJSON)
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, err
		}
	}
	piece, err := json.Marshal(map[string]any{
		"role": role,
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	})
	if err != nil {
		return nil, err
	}
	list = append(list, json.RawMessage(piece))
	return json.Marshal(list)
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
