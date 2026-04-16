package teamcreatetool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// TeamCreate implements tools.Tool for TeamCreateTool.ts.
type TeamCreate struct{}

// New returns a TeamCreate tool.
func New() *TeamCreate { return &TeamCreate{} }

func (t *TeamCreate) Name() string      { return TeamCreateToolName }
func (t *TeamCreate) Aliases() []string { return nil }

// Input mirrors TeamCreateTool.ts Input (strictObject schema).
type Input struct {
	TeamName    string `json:"team_name"`
	Description string `json:"description,omitempty"`
	AgentType   string `json:"agent_type,omitempty"`
}

// Output mirrors TeamCreateTool.ts Output type.
type Output struct {
	TeamName     string `json:"team_name"`
	TeamFilePath string `json:"team_file_path"`
	LeadAgentID  string `json:"lead_agent_id"`
}

// RunContext carries team-creation state callbacks.
// Set OnTeamCreate to wire the actual team file creation + app state management.
type RunContext struct {
	// OnTeamCreate creates the team file, task list dir, registers cleanup, etc.
	// Returns the Output. When nil a stub result is returned.
	OnTeamCreate func(ctx context.Context, in Input) (Output, error)
}

type runCtxKey struct{}

// WithRunContext attaches *RunContext for TeamCreate.Run.
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
// Input: strict JSON { team_name, description?, agent_type? }.
// Output: JSON-encoded Output.
func (t *TeamCreate) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	dec := json.NewDecoder(bytes.NewReader(inputJSON))
	dec.DisallowUnknownFields()
	var in Input
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("teamcreatetool: invalid json: %w", err)
	}
	if in.TeamName == "" {
		return nil, fmt.Errorf("teamcreatetool: team_name is required")
	}

	rc := RunContextFrom(ctx)
	if rc != nil && rc.OnTeamCreate != nil {
		out, err := rc.OnTeamCreate(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("teamcreatetool: team creation failed: %w", err)
		}
		return json.Marshal(out)
	}

	// Stub: return a stub team output. Callers must wire OnTeamCreate for real team infra.
	out := Output{
		TeamName:     in.TeamName,
		TeamFilePath: fmt.Sprintf("~/.claude/teams/%s/config.json", in.TeamName),
		LeadAgentID:  "stub-lead-agent-id",
	}
	return json.Marshal(out)
}

// MapResultForMessagesAPI mirrors mapToolResultToToolResultBlockParam in TeamCreateTool.ts.
// Returns the JSON-stringified output as the tool_result content.
func MapResultForMessagesAPI(outJSON []byte) string {
	return string(outJSON)
}
