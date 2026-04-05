package websearchtool

// Upstream WebSearchTool.ts / UI.tsx / toolLimits.ts
const (
	SearchHint           = "search the web for current information"
	UserFacingName       = "Web Search"
	MaxSearchUses        = 8
	MaxResultSizeChars   = 100_000
	ToolSummaryMaxLength = 50
	PermissionMessage    = "WebSearchTool requires permission."
	// ErrQueryZodMin mirrors typical zod v4 min(2) failure wording for logs/tests.
	ErrQueryZodMin = "websearchtool: query must be at least 2 characters"
)
