package agenttool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// Agent implements tools.Tool for AgentTool.tsx.
type Agent struct{}

// New returns an Agent tool.
func New() *Agent { return &Agent{} }

func (a *Agent) Name() string      { return AgentToolName }
func (a *Agent) Aliases() []string { return []string{LegacyAgentToolName} }

// --- Input types ---

// Input mirrors AgentTool.tsx AgentToolInput (fullInputSchema fields).
type Input struct {
	Description     string  `json:"description"`
	Prompt          string  `json:"prompt"`
	SubagentType    string  `json:"subagent_type,omitempty"`
	Model           string  `json:"model,omitempty"`    // "sonnet"|"opus"|"haiku"
	RunInBackground bool    `json:"run_in_background,omitempty"`
	Name            string  `json:"name,omitempty"`
	TeamName        string  `json:"team_name,omitempty"`
	Mode            string  `json:"mode,omitempty"`      // permission mode
	Isolation       string  `json:"isolation,omitempty"` // "worktree"|"remote"
	Cwd             string  `json:"cwd,omitempty"`
}

// --- Output types ---

// ContentBlock mirrors content block with type "text".
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// UsageServerToolUse mirrors the server_tool_use nested object.
type UsageServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests"`
	WebFetchRequests  int `json:"web_fetch_requests"`
}

// UsageCacheCreation mirrors the cache_creation nested object.
type UsageCacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
}

// Usage mirrors the usage object in agentToolResultSchema.
type Usage struct {
	InputTokens               int                 `json:"input_tokens"`
	OutputTokens              int                 `json:"output_tokens"`
	CacheCreationInputTokens  *int                `json:"cache_creation_input_tokens"`
	CacheReadInputTokens      *int                `json:"cache_read_input_tokens"`
	ServerToolUse             *UsageServerToolUse `json:"server_tool_use"`
	ServiceTier               *string             `json:"service_tier"`
	CacheCreation             *UsageCacheCreation `json:"cache_creation"`
}

// SyncOutput mirrors the sync completed output (agentToolResultSchema + status + prompt).
type SyncOutput struct {
	Status            string         `json:"status"` // "completed"
	AgentID           string         `json:"agentId"`
	AgentType         string         `json:"agentType,omitempty"`
	Content           []ContentBlock `json:"content"`
	TotalToolUseCount int            `json:"totalToolUseCount"`
	TotalDurationMs   int64          `json:"totalDurationMs"`
	TotalTokens       int            `json:"totalTokens"`
	Usage             Usage          `json:"usage"`
	Prompt            string         `json:"prompt"`
}

// AsyncOutput mirrors the async_launched output.
type AsyncOutput struct {
	Status            string `json:"status"` // "async_launched"
	AgentID           string `json:"agentId"`
	Description       string `json:"description"`
	Prompt            string `json:"prompt"`
	OutputFile        string `json:"outputFile"`
	CanReadOutputFile *bool  `json:"canReadOutputFile,omitempty"`
}

// RunContext carries the agent execution callback.
// Set RunAgent to wire the actual subagent query loop execution.
type RunContext struct {
	// RunAgent executes the subagent and returns JSON-encoded SyncOutput or AsyncOutput.
	// When nil, Run returns an error.
	RunAgent func(ctx context.Context, in Input) ([]byte, error)
}

type runCtxKey struct{}

// WithRunContext attaches *RunContext for Agent.Run.
func WithRunContext(ctx context.Context, rc *RunContext) context.Context {
	if rc == nil {
		return ctx
	}
	return context.WithValue(ctx, runCtxKey{}, rc)
}

// RunContextFrom returns *RunContext or nil.
func RunContextFrom(ctx context.Context) *RunContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(runCtxKey{}).(*RunContext)
	return v
}

// Run implements tools.Tool.
// Input: JSON-encoded Input (non-strict; extra fields silently ignored per TS passthrough).
// Output: JSON-encoded SyncOutput or AsyncOutput.
func (a *Agent) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var in Input
	if err := json.NewDecoder(bytes.NewReader(inputJSON)).Decode(&in); err != nil {
		return nil, fmt.Errorf("agenttool: invalid json: %w", err)
	}
	if in.Description == "" {
		return nil, fmt.Errorf("agenttool: description is required")
	}
	if in.Prompt == "" {
		return nil, fmt.Errorf("agenttool: prompt is required")
	}

	rc := RunContextFrom(ctx)
	if rc == nil || rc.RunAgent == nil {
		return nil, fmt.Errorf("agenttool: RunAgent not wired — set RunContext.RunAgent to execute subagents")
	}

	return rc.RunAgent(ctx, in)
}

// MapResultForMessagesAPI mirrors mapToolResultToToolResultBlockParam in AgentTool.tsx.
// Returns the tool_result content for the simpler non-teammate sync path.
// For the full teammate/async paths, callers must implement the branching themselves.
func MapResultForMessagesAPI(outJSON []byte) string {
	// Try to unmarshal as sync output first.
	var sync SyncOutput
	if err := json.Unmarshal(outJSON, &sync); err == nil && sync.Status == "completed" {
		// Extract text content.
		var text string
		for _, c := range sync.Content {
			if c.Type == "text" {
				text += c.Text
			}
		}
		// One-shot agents skip the SendMessage/agentId trailer.
		if OneShotBuiltInAgentTypes[sync.AgentType] {
			return text
		}
		trailer := fmt.Sprintf("\n\n<agent-id>%s</agent-id>", sync.AgentID)
		return text + trailer
	}

	// Async launched path.
	var async AsyncOutput
	if err := json.Unmarshal(outJSON, &async); err == nil && async.Status == "async_launched" {
		canRead := async.CanReadOutputFile != nil && *async.CanReadOutputFile
		if canRead {
			return fmt.Sprintf("Agent launched in background.\nAgent ID: %s\nCheck progress: Read(\"%s\")", async.AgentID, async.OutputFile)
		}
		return fmt.Sprintf("Agent launched in background.\nAgent ID: %s\nOutput file: %s", async.AgentID, async.OutputFile)
	}

	// Fallback: return raw JSON.
	return string(outJSON)
}
