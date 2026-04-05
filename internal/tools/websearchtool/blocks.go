package websearchtool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// MakeOutputFromContentBlocks mirrors makeOutputFromSearchResponse in WebSearchTool.ts.
// Each element of blocks should be one Messages API assistant content block as JSON (type discriminator).
func MakeOutputFromContentBlocks(blocks []json.RawMessage, query string, durationSeconds float64) ([]any, error) {
	var results []any
	textAcc := ""
	inText := true

	flushText := func() {
		if strings.TrimSpace(textAcc) == "" {
			return
		}
		results = append(results, strings.TrimSpace(textAcc))
		textAcc = ""
	}

	for _, raw := range blocks {
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue
		}
		switch probe.Type {
		case "server_tool_use":
			if inText {
				inText = false
				flushText()
			}
		case "web_search_tool_result":
			var wb struct {
				Type      string          `json:"type"`
				ToolUseID string          `json:"tool_use_id"`
				Content   json.RawMessage `json:"content"`
			}
			if err := json.Unmarshal(raw, &wb); err != nil {
				continue
			}
			content := bytes.TrimSpace(wb.Content)
			if len(content) == 0 || string(content) == "null" {
				continue
			}
			if content[0] == '[' {
				var hits []SearchHit
				if err := json.Unmarshal(content, &hits); err != nil {
					continue
				}
				results = append(results, SearchResultBlock{
					ToolUseID: wb.ToolUseID,
					Content:   hits,
				})
				continue
			}
			var errObj struct {
				ErrorCode string `json:"error_code"`
			}
			_ = json.Unmarshal(content, &errObj)
			code := strings.TrimSpace(errObj.ErrorCode)
			if code == "" {
				code = "unknown"
			}
			results = append(results, fmt.Sprintf("Web search error: %s", code))
		case "text":
			var tb struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(raw, &tb); err != nil {
				continue
			}
			if inText {
				textAcc += tb.Text
			} else {
				inText = true
				textAcc = tb.Text
			}
		}
	}
	if strings.TrimSpace(textAcc) != "" {
		results = append(results, strings.TrimSpace(textAcc))
	}
	return results, nil
}
