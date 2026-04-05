package webfetchtool

import "encoding/json"

// MapWebFetchToolResultForMessagesAPI mirrors WebFetchTool.mapToolResultToToolResultBlockParam.
func MapWebFetchToolResultForMessagesAPI(outJSON []byte) string {
	var m struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(outJSON, &m); err != nil || m.Result == "" {
		return ""
	}
	return m.Result
}
