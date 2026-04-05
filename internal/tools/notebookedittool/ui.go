package notebookedittool

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Upstream: NotebookEditTool.ts mapToolResultToToolResultBlockParam (string for Messages API).
// UI.tsx supplies TUI render paths only; headless parity uses this mapping.

// MapNotebookEditToolResultForMessagesAPI mirrors NotebookEditTool.mapToolResultToToolResultBlockParam.
func MapNotebookEditToolResultForMessagesAPI(outJSON []byte) string {
	var m struct {
		CellID   string `json:"cell_id"`
		EditMode string `json:"edit_mode"`
		NewSrc   string `json:"new_source"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(outJSON, &m); err != nil {
		return ""
	}
	if strings.TrimSpace(m.Error) != "" {
		return m.Error
	}
	switch m.EditMode {
	case "replace":
		return fmt.Sprintf("Updated cell %s with %s", m.CellID, m.NewSrc)
	case "insert":
		return fmt.Sprintf("Inserted cell %s with %s", m.CellID, m.NewSrc)
	case "delete":
		return fmt.Sprintf("Deleted cell %s", m.CellID)
	default:
		return "Unknown edit mode"
	}
}
