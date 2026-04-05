package websearchtool

import (
	"encoding/json"
	"regexp"
)

// WebSearchProgress mirrors ProgressMessage<WebSearchProgress> / onProgress payloads in WebSearchTool.ts.
type WebSearchProgress struct {
	ToolUseID string                `json:"toolUseID"`
	Data      WebSearchProgressData `json:"data"`
}

// WebSearchProgressData is the discriminated `data` object (query_update | search_results_received).
type WebSearchProgressData struct {
	Type        string `json:"type"`
	Query       string `json:"query,omitempty"`
	ResultCount int    `json:"resultCount,omitempty"`
}

var webSearchPartialQueryRe = regexp.MustCompile(`"query"\s*:\s*"((?:[^"\\]|\\.)*)"`)

// ExtractQueryFromPartialWebSearchInputJSON mirrors the regex + jsonParse path in WebSearchTool.call
// for input_json_delta fragments on server_tool_use.
func ExtractQueryFromPartialWebSearchInputJSON(s string) (query string, ok bool) {
	m := webSearchPartialQueryRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return "", false
	}
	inner := `"` + m[1] + `"`
	var q string
	if err := json.Unmarshal([]byte(inner), &q); err != nil {
		return "", false
	}
	return q, true
}
