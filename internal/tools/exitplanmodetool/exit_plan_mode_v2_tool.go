package exitplanmodetool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ExitPlanModeV2 implements tools.Tool for ExitPlanModeV2Tool.ts.
type ExitPlanModeV2 struct{}

// New returns an ExitPlanModeV2 tool.
func New() *ExitPlanModeV2 { return &ExitPlanModeV2{} }

func (e *ExitPlanModeV2) Name() string      { return ExitPlanModeToolName }
func (e *ExitPlanModeV2) Aliases() []string { return nil }

// AllowedPrompt mirrors AllowedPrompt in ExitPlanModeV2Tool.ts (allowedPromptSchema).
type AllowedPrompt struct {
	Tool   string `json:"tool"`
	Prompt string `json:"prompt"`
}

// exitPlanModeInput mirrors the inputSchema in ExitPlanModeV2Tool.ts.
// The schema uses .passthrough() so extra fields are allowed.
type exitPlanModeInput struct {
	AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitempty"`
}

// Output mirrors ExitPlanModeV2Tool.ts export type Output (outputSchema fields).
type Output struct {
	Plan                   *string `json:"plan"`
	IsAgent                bool    `json:"isAgent"`
	FilePath               string  `json:"filePath,omitempty"`
	HasTaskTool            *bool   `json:"hasTaskTool,omitempty"`
	PlanWasEdited          *bool   `json:"planWasEdited,omitempty"`
	AwaitingLeaderApproval *bool   `json:"awaitingLeaderApproval,omitempty"`
	RequestID              string  `json:"requestId,omitempty"`
}

// RunContext carries plan-mode exit state callbacks.
// Set OnExitPlanMode to wire the full plan-file read + permission restoration pipeline
// (getPlan, setAppState, setHasExitedPlanMode in TS).
type RunContext struct {
	// AgentID is non-empty when the tool is called from a sub-agent context.
	AgentID string
	// OnExitPlanMode is called after input validation to execute the plan-mode exit.
	// Returns the Output that will be JSON-encoded. When nil a stub approved output is returned.
	OnExitPlanMode func(ctx context.Context, allowedPrompts []AllowedPrompt) (Output, error)
}

type runCtxKey struct{}

// WithRunContext attaches *RunContext for ExitPlanModeV2.Run.
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
// Input: { allowedPrompts?: [...] } with passthrough (extra fields allowed).
// Output: JSON-encoded Output.
func (e *ExitPlanModeV2) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Note: TS uses .passthrough() so unknown fields are allowed. We use a lenient decoder.
	var in exitPlanModeInput
	if err := json.NewDecoder(bytes.NewReader(inputJSON)).Decode(&in); err != nil {
		return nil, fmt.Errorf("exitplanmodetool: invalid json: %w", err)
	}

	rc := RunContextFrom(ctx)
	if rc != nil && rc.OnExitPlanMode != nil {
		out, err := rc.OnExitPlanMode(ctx, in.AllowedPrompts)
		if err != nil {
			return nil, fmt.Errorf("exitplanmodetool: plan exit failed: %w", err)
		}
		return json.Marshal(out)
	}

	// Stub: return an approved non-agent plan result without plan content.
	// Callers must wire OnExitPlanMode for the full plan-file + permission flow.
	out := Output{
		Plan:    nil,
		IsAgent: false,
	}
	return json.Marshal(out)
}

// MapResultForMessagesAPI mirrors mapToolResultToToolResultBlockParam in ExitPlanModeV2Tool.ts.
// It returns the tool_result content string for the given output JSON.
func MapResultForMessagesAPI(outJSON []byte) string {
	var o Output
	if err := json.Unmarshal(outJSON, &o); err != nil {
		return "User has approved exiting plan mode. You can now proceed."
	}

	// Teammate awaiting leader approval path.
	if o.AwaitingLeaderApproval != nil && *o.AwaitingLeaderApproval {
		filePath := o.FilePath
		requestID := o.RequestID
		return fmt.Sprintf(`Your plan has been submitted to the team lead for approval.

Plan file: %s

**What happens next:**
1. Wait for the team lead to review your plan
2. You will receive a message in your inbox with approval/rejection
3. If approved, you can proceed with implementation
4. If rejected, refine your plan based on the feedback

**Important:** Do NOT proceed until you receive approval. Check your inbox for response.

Request ID: %s`, filePath, requestID)
	}

	// Sub-agent (isAgent) path.
	if o.IsAgent {
		return "User has approved the plan. There is nothing else needed from you now. Please respond with \"ok\""
	}

	// Empty plan path.
	plan := ""
	if o.Plan != nil {
		plan = *o.Plan
	}
	if strings.TrimSpace(plan) == "" {
		return "User has approved exiting plan mode. You can now proceed."
	}

	// Normal path: echo plan with optional team hint.
	planLabel := "Approved Plan"
	if o.PlanWasEdited != nil && *o.PlanWasEdited {
		planLabel = "Approved Plan (edited by user)"
	}
	filePath := o.FilePath
	return fmt.Sprintf(`User has approved your plan. You can now start coding. Start with updating your todo list if applicable

Your plan has been saved to: %s
You can refer back to it if needed during implementation.

## %s:
%s`, filePath, planLabel, plan)
}
