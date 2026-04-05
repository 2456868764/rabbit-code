package websearchtool

// Upstream WebSearchTool.call / claude.ts query options (string parity for future streaming wiring).
const (
	// QuerySourceWebSearchTool is options.querySource in WebSearchTool.call.
	QuerySourceWebSearchTool = "web_search_tool"
	// InnerSearchSystemPrompt is the single system line for the inner streaming request.
	InnerSearchSystemPrompt = "You are an assistant for performing a web search tool use"
	// ServerToolSchemaName is BetaWebSearchTool20250305.name ("web_search", not "WebSearch").
	ServerToolSchemaName = "web_search"
	// ToolShouldDefer mirrors shouldDefer: true in buildTool.
	ToolShouldDefer = true
	// PermissionSuggestionType mirrors checkPermissions suggestions[0].type.
	PermissionSuggestionType = "addRules"
	// PermissionSuggestionDestination mirrors destination: 'localSettings'.
	PermissionSuggestionDestination = "localSettings"
	// PermissionSuggestionBehavior mirrors behavior: 'allow'.
	PermissionSuggestionBehavior = "allow"
)

// InnerSearchUserContent returns the user message content for the inner search request (createUserMessage).
func InnerSearchUserContent(query string) string {
	return "Perform a web search for the query: " + query
}

// AutoClassifierInput mirrors toAutoClassifierInput (query-only).
func AutoClassifierInput(in Input) string {
	return in.Query
}

// ExtractSearchText mirrors extractSearchText() — always empty (UI chrome only).
func ExtractSearchText() string { return "" }

// PermissionRule mirrors suggestions[].rules[] entry.
type PermissionRule struct {
	ToolName string `json:"toolName"`
}

// PermissionSuggestion mirrors checkPermissions suggestions[0].
type PermissionSuggestion struct {
	Type        string           `json:"type"`
	Rules       []PermissionRule `json:"rules"`
	Behavior    string           `json:"behavior"`
	Destination string           `json:"destination"`
}

// DefaultPermissionSuggestions mirrors WebSearchTool.checkPermissions suggestions slice.
func DefaultPermissionSuggestions() []PermissionSuggestion {
	return []PermissionSuggestion{{
		Type:        PermissionSuggestionType,
		Rules:       []PermissionRule{{ToolName: WebSearchToolName}},
		Behavior:    PermissionSuggestionBehavior,
		Destination: PermissionSuggestionDestination,
	}}
}

// CheckPermissionsResult mirrors PermissionResult passthrough payload (headless).
type CheckPermissionsResult struct {
	Behavior    string
	Message     string
	Suggestions []PermissionSuggestion
}

// DefaultCheckPermissions mirrors checkPermissions(_input) return value.
func DefaultCheckPermissions() CheckPermissionsResult {
	return CheckPermissionsResult{
		Behavior:    "passthrough",
		Message:     PermissionMessage,
		Suggestions: DefaultPermissionSuggestions(),
	}
}
