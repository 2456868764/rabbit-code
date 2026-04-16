package teamdeletetool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// TeamDelete implements tools.Tool for TeamDeleteTool.ts.
type TeamDelete struct{}

// New returns a TeamDelete tool.
func New() *TeamDelete { return &TeamDelete{} }

func (t *TeamDelete) Name() string      { return TeamDeleteToolName }
func (t *TeamDelete) Aliases() []string { return nil }

// Output mirrors TeamDeleteTool.ts Output type.
type Output struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	TeamName string `json:"team_name,omitempty"`
}

// RunContext carries team-deletion state callbacks.
// Set OnTeamDelete to wire actual team file cleanup + app state management.
type RunContext struct {
	// TeamName is the current team name (from app state teamContext.teamName).
	TeamName string
	// OnTeamDelete performs the actual cleanup (readTeamFile / cleanupTeamDirectories /
	// unregisterTeamForSessionCleanup / clearTeammateColors / clearLeaderTeamName).
	// When nil a stub result is returned.
	OnTeamDelete func(ctx context.Context, teamName string) (Output, error)
}

type runCtxKey struct{}

// WithRunContext attaches *RunContext for TeamDelete.Run.
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
// Input: strict empty JSON object {}.
// Output: JSON-encoded Output.
func (t *TeamDelete) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	dec := json.NewDecoder(bytes.NewReader(inputJSON))
	dec.DisallowUnknownFields()
	var in struct{}
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("teamdeletetool: invalid json: %w", err)
	}

	rc := RunContextFrom(ctx)
	teamName := ""
	if rc != nil {
		teamName = rc.TeamName
	}

	if rc != nil && rc.OnTeamDelete != nil {
		out, err := rc.OnTeamDelete(ctx, teamName)
		if err != nil {
			return nil, fmt.Errorf("teamdeletetool: team deletion failed: %w", err)
		}
		return json.Marshal(out)
	}

	// Stub: return success when no team context is set.
	out := Output{
		Success:  true,
		Message:  "Team deleted successfully.",
		TeamName: teamName,
	}
	return json.Marshal(out)
}

// MapResultForMessagesAPI mirrors mapToolResultToToolResultBlockParam in TeamDeleteTool.ts.
// Returns the JSON-stringified output as the tool_result content.
func MapResultForMessagesAPI(outJSON []byte) string {
	return string(outJSON)
}
