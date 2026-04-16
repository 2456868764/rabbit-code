// Package agenttool implements the Agent tool (P6.5).
package agenttool

// AgentToolName is AGENT_TOOL_NAME upstream.
const AgentToolName = "Agent"

// LegacyAgentToolName is LEGACY_AGENT_TOOL_NAME — backward compat alias for "Task".
const LegacyAgentToolName = "Task"

// VerificationAgentType is VERIFICATION_AGENT_TYPE upstream.
const VerificationAgentType = "verification"

// OneShotBuiltInAgentTypes mirrors ONE_SHOT_BUILTIN_AGENT_TYPES — agents that run
// once and return a report without SendMessage trailers.
var OneShotBuiltInAgentTypes = map[string]bool{
	"Explore": true,
	"Plan":    true,
}

// Tool metadata constants from AgentTool.tsx buildTool definition.
const (
	// SearchHint is not explicitly set in TS but defaults to tool description.
	SearchHint = "launch a subagent to perform a task"
	// MaxResultSizeChars mirrors maxResultSizeChars (not set, defaults to 100_000).
	MaxResultSizeChars = 100_000
	// ShouldDefer mirrors shouldDefer: true in AgentTool.tsx.
	ShouldDefer = true
)
