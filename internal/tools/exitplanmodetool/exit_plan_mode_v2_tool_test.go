package exitplanmodetool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestExitPlanModeV2_stubRun(t *testing.T) {
	out, err := New().Run(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.IsAgent {
		t.Fatal("stub should not set isAgent")
	}
}

func TestExitPlanModeV2_allowedPrompts(t *testing.T) {
	raw := `{"allowedPrompts":[{"tool":"Bash","prompt":"run tests"}]}`
	out, err := New().Run(context.Background(), []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
}

func TestExitPlanModeV2_onExitCalled(t *testing.T) {
	called := false
	plan := "step 1\nstep 2"
	rc := &RunContext{
		OnExitPlanMode: func(_ context.Context, _ []AllowedPrompt) (Output, error) {
			called = true
			return Output{Plan: &plan, IsAgent: false, FilePath: "/tmp/plan.md"}, nil
		},
	}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("OnExitPlanMode not called")
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.Plan == nil || *o.Plan != plan {
		t.Fatalf("unexpected plan: %v", o.Plan)
	}
}

func TestExitPlanModeV2_passthroughExtraFields(t *testing.T) {
	// .passthrough() in TS means extra fields are allowed (no error).
	_, err := New().Run(context.Background(), []byte(`{"extraField":"value","allowedPrompts":[]}`))
	if err != nil {
		t.Fatalf("passthrough should allow extra fields: %v", err)
	}
}

func TestMapResultForMessagesAPI_emptyPlan(t *testing.T) {
	raw, _ := json.Marshal(Output{Plan: nil, IsAgent: false})
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "approved") {
		t.Fatalf("unexpected: %q", msg)
	}
}

func TestMapResultForMessagesAPI_normalPlan(t *testing.T) {
	plan := "1. Do this\n2. Do that"
	raw, _ := json.Marshal(Output{Plan: &plan, IsAgent: false, FilePath: "/plan.md"})
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "start coding") {
		t.Fatalf("unexpected: %q", msg)
	}
	if !strings.Contains(msg, plan) {
		t.Fatalf("plan not in message: %q", msg)
	}
}

func TestMapResultForMessagesAPI_agentPath(t *testing.T) {
	raw, _ := json.Marshal(Output{Plan: ptr("some plan"), IsAgent: true})
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "approved the plan") {
		t.Fatalf("unexpected: %q", msg)
	}
}

func TestMapResultForMessagesAPI_awaitingLeaderApproval(t *testing.T) {
	raw, _ := json.Marshal(Output{
		Plan:                   ptr("plan content"),
		IsAgent:                true,
		FilePath:               "/plans/p.md",
		AwaitingLeaderApproval: ptr(true),
		RequestID:              "req-123",
	})
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "team lead") {
		t.Fatalf("unexpected: %q", msg)
	}
	if !strings.Contains(msg, "req-123") {
		t.Fatalf("requestId not in message: %q", msg)
	}
}

func TestMapResultForMessagesAPI_planWasEdited(t *testing.T) {
	plan := "edited plan"
	raw, _ := json.Marshal(Output{Plan: &plan, IsAgent: false, FilePath: "/p.md", PlanWasEdited: ptr(true)})
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "edited by user") {
		t.Fatalf("expected edited label: %q", msg)
	}
}

func TestExitPlanModeToolName(t *testing.T) {
	if ExitPlanModeToolName != "ExitPlanMode" {
		t.Fatalf("unexpected: %q", ExitPlanModeToolName)
	}
	if New().Name() != ExitPlanModeToolName {
		t.Fatal("Name() mismatch")
	}
}
