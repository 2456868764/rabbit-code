package query

import (
	"encoding/json"
	"strings"

	"github.com/2456868764/rabbit-code/internal/utils/messages"
)

// TodoResumeItem mirrors utils/todo/types.ts TodoItem (sessionRestore.ts extractTodosFromTranscript).
type TodoResumeItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

// ExtractTodosFromTranscriptJSON scans Messages API JSON (newest assistant first) for the last TodoWrite tool_use
// and returns its todos array (utils/sessionRestore.ts extractTodosFromTranscript).
func ExtractTodosFromTranscriptJSON(transcript []byte) ([]TodoResumeItem, error) {
	var rawMsgs []json.RawMessage
	if err := json.Unmarshal(transcript, &rawMsgs); err != nil {
		return nil, err
	}
	for i := len(rawMsgs) - 1; i >= 0; i-- {
		var env struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(rawMsgs[i], &env); err != nil {
			continue
		}
		if strings.TrimSpace(env.Role) != "assistant" {
			continue
		}
		if todos, ok := todosFromAssistantContent(env.Content); ok {
			return todos, nil
		}
	}
	return nil, nil
}

func todosFromAssistantContent(content json.RawMessage) ([]TodoResumeItem, bool) {
	if len(content) == 0 {
		return nil, false
	}
	if content[0] == '"' {
		return nil, false
	}
	var blocks []struct {
		Type  string          `json:"type"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil, false
	}
	for j := len(blocks) - 1; j >= 0; j-- {
		b := blocks[j]
		if strings.TrimSpace(b.Type) != "tool_use" {
			continue
		}
		if strings.TrimSpace(b.Name) != messages.ToolNameTodoWrite {
			continue
		}
		var input struct {
			Todos json.RawMessage `json:"todos"`
		}
		if err := json.Unmarshal(b.Input, &input); err != nil || len(input.Todos) == 0 {
			return nil, true
		}
		var list []TodoResumeItem
		if err := json.Unmarshal(input.Todos, &list); err != nil {
			return nil, true
		}
		out := make([]TodoResumeItem, 0, len(list))
		for _, t := range list {
			if strings.TrimSpace(t.Content) == "" || strings.TrimSpace(t.ActiveForm) == "" {
				continue
			}
			switch strings.TrimSpace(t.Status) {
			case "pending", "in_progress", "completed":
			default:
				continue
			}
			out = append(out, t)
		}
		return out, true
	}
	return nil, false
}
