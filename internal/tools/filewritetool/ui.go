package filewritetool

import (
	"encoding/json"
	"fmt"
)

// Upstream: FileWriteTool/UI.tsx presentation; string mapping matches FileWriteTool.mapToolResultToToolResultBlockParam.
//
// MapWriteToolResultForMessagesAPI mirrors FileWriteTool.mapToolResultToToolResultBlockParam (string tool_result for transcript).
func MapWriteToolResultForMessagesAPI(outJSON []byte) string {
	var m struct {
		Type     string `json:"type"`
		FilePath string `json:"filePath"`
	}
	if err := json.Unmarshal(outJSON, &m); err != nil {
		return ""
	}
	switch m.Type {
	case "create":
		return fmt.Sprintf("File created successfully at: %s", m.FilePath)
	case "update":
		return fmt.Sprintf("The file %s has been updated successfully.", m.FilePath)
	default:
		return ""
	}
}
