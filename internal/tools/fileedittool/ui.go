package fileedittool

import (
	"encoding/json"
	"fmt"
)

// Upstream: FileEditTool/UI.tsx presentation; string mapping matches FileEditTool.mapToolResultToToolResultBlockParam.
//
// MapEditToolResultForMessagesAPI mirrors FileEditTool.mapToolResultToToolResultBlockParam.
func MapEditToolResultForMessagesAPI(outJSON []byte) string {
	var m struct {
		FilePath     string `json:"filePath"`
		UserModified bool   `json:"userModified"`
		ReplaceAll   bool   `json:"replaceAll"`
	}
	if err := json.Unmarshal(outJSON, &m); err != nil {
		return ""
	}
	modNote := ""
	if m.UserModified {
		modNote = ".  The user modified your proposed changes before accepting them. "
	}
	if m.ReplaceAll {
		return fmt.Sprintf("The file %s has been updated%s. All occurrences were successfully replaced.", m.FilePath, modNote)
	}
	return fmt.Sprintf("The file %s has been updated successfully%s.", m.FilePath, modNote)
}
