package bashtool

import (
	"encoding/json"
	"strings"
)

// MapBashToolResultForMessagesAPI mirrors BashTool.mapToolResultToToolResultBlockParam (text path; structuredContent / isImage / persistedOutput defer).
func MapBashToolResultForMessagesAPI(outJSON []byte) string {
	var o struct {
		Stdout      string `json:"stdout"`
		Stderr      string `json:"stderr"`
		Interrupted bool   `json:"interrupted"`
	}
	if err := json.Unmarshal(outJSON, &o); err != nil {
		return string(outJSON)
	}
	processed := o.Stdout
	if processed != "" {
		processed = strings.TrimLeft(processed, " \t\n\r")
		for strings.HasPrefix(processed, "\n") {
			processed = strings.TrimPrefix(processed, "\n")
			processed = strings.TrimLeft(processed, " \t\n\r")
		}
		processed = strings.TrimRight(processed, " \t\n\r")
	}
	errMsg := strings.TrimSpace(o.Stderr)
	if o.Interrupted {
		if errMsg != "" {
			errMsg += "\n"
		}
		errMsg += "<error>Command was aborted before completion</error>"
	}
	parts := []string{}
	if processed != "" {
		parts = append(parts, processed)
	}
	if errMsg != "" {
		parts = append(parts, errMsg)
	}
	return strings.Join(parts, "\n")
}
