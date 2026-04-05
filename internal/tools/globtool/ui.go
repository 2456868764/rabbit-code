package globtool

import (
	"encoding/json"
	"strings"
)

// MapGlobToolResultForMessagesAPI mirrors GlobTool.ts mapToolResultToToolResultBlockParam (headless string / lines).
func MapGlobToolResultForMessagesAPI(out []byte) string {
	var o struct {
		Filenames []string `json:"filenames"`
		Truncated bool     `json:"truncated"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		return ""
	}
	if len(o.Filenames) == 0 {
		return "No files found"
	}
	lines := append([]string{}, o.Filenames...)
	if o.Truncated {
		lines = append(lines, "(Results are truncated. Consider using a more specific path or pattern.)")
	}
	return strings.Join(lines, "\n")
}
