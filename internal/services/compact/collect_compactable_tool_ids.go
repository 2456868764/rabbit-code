package compact

import "encoding/json"

// CollectCompactableToolUseIDsFromTranscriptJSON returns tool_use ids for compactable tools in assistant
// messages (microCompact.ts collectCompactableToolIds; transcript = API messages JSON array).
func CollectCompactableToolUseIDsFromTranscriptJSON(transcript []byte) ([]string, error) {
	var arr []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return nil, err
	}
	var ids []string
	for _, m := range arr {
		if m.Role != "assistant" || len(m.Content) == 0 {
			continue
		}
		var blocks []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(m.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_use" && b.ID != "" && IsCompactableToolName(b.Name) {
				ids = append(ids, b.ID)
			}
		}
	}
	return ids, nil
}
