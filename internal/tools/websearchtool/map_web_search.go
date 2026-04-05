package websearchtool

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// MapWebSearchToolResultForMessagesAPI mirrors WebSearchTool.mapToolResultToToolResultBlockParam.
func MapWebSearchToolResultForMessagesAPI(outJSON []byte) string {
	var payload struct {
		Query   string            `json:"query"`
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(outJSON, &payload); err != nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(`Web search results for query: "`)
	b.WriteString(payload.Query)
	b.WriteString("\"\n\n")

	for _, raw := range payload.Results {
		if raw == nil || len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		raw = bytes.TrimSpace(raw)
		if isJSONNull(raw) {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			b.WriteString(s)
			b.WriteString("\n\n")
			continue
		}
		var obj struct {
			Content []struct {
				Title string `json:"title"`
				URL   string `json:"url"`
			} `json:"content"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		if len(obj.Content) > 0 {
			links, _ := json.Marshal(obj.Content)
			b.WriteString("Links: ")
			b.Write(links)
			b.WriteString("\n\n")
		} else {
			b.WriteString("No links found.\n\n")
		}
	}

	b.WriteString("\nREMINDER: You MUST include the sources above in your response to the user using markdown hyperlinks.")
	formatted := strings.TrimSpace(b.String())
	return trimToolResultToMaxRunes(formatted, MaxResultSizeChars)
}

func trimToolResultToMaxRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "\n…(truncated)"
}

func isJSONNull(raw []byte) bool {
	return len(raw) == 4 && string(raw) == "null"
}
