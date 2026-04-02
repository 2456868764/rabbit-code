package query

import (
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
