package enterplanmodetool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestEnterPlanMode_basicRun(t *testing.T) {
	out, err := New().Run(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(o.Message, "plan mode") {
		t.Fatalf("unexpected message: %q", o.Message)
	}
}

func TestEnterPlanMode_rejectsUnknownFields(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"extra":1}`))
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestEnterPlanMode_rejectsAgentContext(t *testing.T) {
	ctx := WithRunContext(context.Background(), &RunContext{AgentID: "sub-agent-1"})
	_, err := New().Run(ctx, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for agent context")
	}
	if !strings.Contains(err.Error(), "agent") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnterPlanMode_callsOnEnterPlanMode(t *testing.T) {
	called := false
	rc := &RunContext{
		OnEnterPlanMode: func(_ context.Context) error {
			called = true
			return nil
		},
	}
	ctx := WithRunContext(context.Background(), rc)
	if _, err := New().Run(ctx, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("OnEnterPlanMode was not called")
	}
}

func TestMapResultForMessagesAPI_containsInstructions(t *testing.T) {
	raw, _ := json.Marshal(Output{Message: defaultPlanModeMessage})
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "ExitPlanMode") {
		t.Fatalf("expected instructions in message, got: %q", msg)
	}
	if !strings.Contains(msg, "DO NOT write") {
		t.Fatalf("expected read-only warning, got: %q", msg)
	}
}

func TestEnterPlanModeToolName(t *testing.T) {
	if EnterPlanModeToolName != "EnterPlanMode" {
		t.Fatalf("unexpected tool name: %q", EnterPlanModeToolName)
	}
	if New().Name() != EnterPlanModeToolName {
		t.Fatal("Name() mismatch")
	}
}

func TestPromptNotEmpty(t *testing.T) {
	p := GetEnterPlanModeToolPrompt()
	if len(p) < 100 {
		t.Fatalf("prompt too short: %q", p)
	}
}
