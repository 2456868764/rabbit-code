package enterplanmodetool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// EnterPlanMode implements tools.Tool for EnterPlanModeTool.ts.
type EnterPlanMode struct{}

// New returns an EnterPlanMode tool.
func New() *EnterPlanMode { return &EnterPlanMode{} }

func (e *EnterPlanMode) Name() string        { return EnterPlanModeToolName }
func (e *EnterPlanMode) Aliases() []string   { return nil }

// Output mirrors EnterPlanModeTool.ts export type Output (outputSchema fields).
type Output struct {
	Message string `json:"message"`
}

// RunContext carries plan-mode state callbacks for the tool.
// Set OnEnterPlanMode to wire the actual permission-mode transition when the
// full engine is available (handlePlanModeTransition + setAppState in TS).
type RunContext struct {
	// AgentID is non-empty when the tool is called from a sub-agent context
	// (EnterPlanMode is forbidden for agents in TS).
	AgentID string
	// OnEnterPlanMode is called after input validation to execute the plan-mode
	// transition. When nil the tool returns a stub result without modifying state.
	OnEnterPlanMode func(ctx context.Context) error
}

type runCtxKey struct{}

// WithRunContext attaches *RunContext for EnterPlanMode.Run.
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

// defaultPlanModeMessage mirrors the literal returned by EnterPlanModeTool.ts call().
const defaultPlanModeMessage = "Entered plan mode. You should now focus on exploring the codebase and designing an implementation approach."

// planModeInstructions mirrors mapToolResultToToolResultBlockParam in EnterPlanModeTool.ts
// (isPlanModeInterviewPhaseEnabled() == false path).
const planModeInstructions = `

In plan mode, you should:
1. Thoroughly explore the codebase to understand existing patterns
2. Identify similar features and architectural approaches
3. Consider multiple approaches and their trade-offs
4. Use AskUserQuestion if you need to clarify the approach
5. Design a concrete implementation strategy
6. When ready, use ExitPlanMode to present your plan for approval

Remember: DO NOT write or edit any files yet. This is a read-only exploration and planning phase.`

// MapResultForMessagesAPI mirrors mapToolResultToToolResultBlockParam in EnterPlanModeTool.ts.
// It returns the content string for a tool_result block.
func MapResultForMessagesAPI(outJSON []byte) string {
	var o Output
	if err := json.Unmarshal(outJSON, &o); err != nil || strings.TrimSpace(o.Message) == "" {
		return defaultPlanModeMessage + planModeInstructions
	}
	return o.Message + planModeInstructions
}

// Run implements tools.Tool.
// Input: strict empty JSON object `{}` (no parameters per TS inputSchema).
// Output: JSON-encoded Output{Message}.
func (e *EnterPlanMode) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	dec := json.NewDecoder(bytes.NewReader(inputJSON))
	dec.DisallowUnknownFields()
	var in struct{}
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("enterplanmodetool: invalid json: %w", err)
	}

	rc := RunContextFrom(ctx)
	if rc != nil && strings.TrimSpace(rc.AgentID) != "" {
		return nil, errors.New("enterplanmodetool: EnterPlanMode tool cannot be used in agent contexts")
	}

	if rc != nil && rc.OnEnterPlanMode != nil {
		if err := rc.OnEnterPlanMode(ctx); err != nil {
			return nil, fmt.Errorf("enterplanmodetool: plan mode transition failed: %w", err)
		}
	}

	out := Output{Message: defaultPlanModeMessage}
	return json.Marshal(out)
}
