package websearchtool

// WebSearchToolSchema20250305 mirrors BetaWebSearchTool20250305 / makeToolSchema in WebSearchTool.ts.
type WebSearchToolSchema20250305 struct {
	Type           string   `json:"type"`
	Name           string   `json:"name"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
	MaxUses        int      `json:"max_uses"`
}

// WebSearchToolSchemaFromInput builds the extra_tool_schema entry (max_uses fixed at 8 upstream).
func WebSearchToolSchemaFromInput(in Input) WebSearchToolSchema20250305 {
	return WebSearchToolSchema20250305{
		Type:           "web_search_20250305",
		Name:           ServerToolSchemaName,
		AllowedDomains: in.AllowedDomains,
		BlockedDomains: in.BlockedDomains,
		MaxUses:        MaxSearchUses,
	}
}
