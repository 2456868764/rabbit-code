package toolsearchtool

import (
	"encoding/json"
	"strings"
)

// MapToolSearchToolResultForMessagesAPI mirrors ToolSearchTool.mapToolResultToToolResultBlockParam.
// Returns either a string (no matches) or []any of tool_reference-shaped maps.
func MapToolSearchToolResultForMessagesAPI(outJSON []byte) any {
	var m struct {
		Matches              []string `json:"matches"`
		PendingMCPServers    []string `json:"pending_mcp_servers"`
	}
	if err := json.Unmarshal(outJSON, &m); err != nil {
		return string(outJSON)
	}
	if len(m.Matches) == 0 {
		text := "No matching deferred tools found"
		if len(m.PendingMCPServers) > 0 {
			text += ". Some MCP servers are still connecting: " + strings.Join(m.PendingMCPServers, ", ") +
				". Their tools will become available shortly — try searching again."
		}
		return text
	}
	refs := make([]any, 0, len(m.Matches))
	for _, name := range m.Matches {
		refs = append(refs, map[string]any{
			"type":      "tool_reference",
			"tool_name": name,
		})
	}
	return refs
}
