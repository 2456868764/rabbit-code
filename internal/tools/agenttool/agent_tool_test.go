package agenttool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAgent_requiresRunAgent(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"description":"do it","prompt":"run tests"}`))
	if err == nil {
		t.Fatal("expected error when RunAgent not wired")
	}
	if !strings.Contains(err.Error(), "RunAgent not wired") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgent_validatesRequiredFields(t *testing.T) {
	rc := &RunContext{RunAgent: func(_ context.Context, _ Input) ([]byte, error) {
		return json.Marshal(SyncOutput{Status: "completed"})
	}}
	ctx := WithRunContext(context.Background(), rc)

	_, err := New().Run(ctx, []byte(`{"description":"","prompt":"x"}`))
	if err == nil {
		t.Fatal("expected error for empty description")
	}

	_, err = New().Run(ctx, []byte(`{"description":"d","prompt":""}`))
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
}

func TestAgent_delegatesToRunAgent(t *testing.T) {
	called := false
	want := SyncOutput{
		Status:  "completed",
		AgentID: "agent-123",
		Content: []ContentBlock{{Type: "text", Text: "done"}},
	}
	rc := &RunContext{RunAgent: func(_ context.Context, in Input) ([]byte, error) {
		called = true
		if in.Description != "test task" {
			t.Errorf("unexpected description: %s", in.Description)
		}
		return json.Marshal(want)
	}}
	ctx := WithRunContext(context.Background(), rc)
	raw, err := New().Run(ctx, []byte(`{"description":"test task","prompt":"do the test"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("RunAgent not called")
	}
	var got SyncOutput
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.AgentID != "agent-123" {
		t.Fatalf("unexpected agentId: %s", got.AgentID)
	}
}

func TestAgent_nameAndAliases(t *testing.T) {
	a := New()
	if a.Name() != "Agent" {
		t.Fatalf("unexpected name: %s", a.Name())
	}
	if len(a.Aliases()) != 1 || a.Aliases()[0] != "Task" {
		t.Fatalf("unexpected aliases: %v", a.Aliases())
	}
}

func TestMapResultForMessagesAPI_sync(t *testing.T) {
	out := SyncOutput{
		Status:  "completed",
		AgentID: "ag-1",
		Content: []ContentBlock{{Type: "text", Text: "result text"}},
	}
	raw, _ := json.Marshal(out)
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "result text") {
		t.Fatalf("unexpected: %q", msg)
	}
	if !strings.Contains(msg, "ag-1") {
		t.Fatalf("agentId not in message: %q", msg)
	}
}

func TestMapResultForMessagesAPI_oneShotSkipsTrailer(t *testing.T) {
	out := SyncOutput{
		Status:    "completed",
		AgentID:   "ag-2",
		AgentType: "Explore",
		Content:   []ContentBlock{{Type: "text", Text: "explored"}},
	}
	raw, _ := json.Marshal(out)
	msg := MapResultForMessagesAPI(raw)
	if strings.Contains(msg, "ag-2") {
		t.Fatalf("one-shot agent should skip agentId trailer: %q", msg)
	}
}

func TestMapResultForMessagesAPI_async(t *testing.T) {
	canRead := true
	out := AsyncOutput{
		Status:            "async_launched",
		AgentID:           "ag-3",
		OutputFile:        "/tmp/out.json",
		CanReadOutputFile: &canRead,
	}
	raw, _ := json.Marshal(out)
	msg := MapResultForMessagesAPI(raw)
	if !strings.Contains(msg, "background") {
		t.Fatalf("unexpected: %q", msg)
	}
	if !strings.Contains(msg, "/tmp/out.json") {
		t.Fatalf("output file not in message: %q", msg)
	}
}

func TestOneShotBuiltInAgentTypes(t *testing.T) {
	if !OneShotBuiltInAgentTypes["Explore"] {
		t.Fatal("Explore should be one-shot")
	}
	if !OneShotBuiltInAgentTypes["Plan"] {
		t.Fatal("Plan should be one-shot")
	}
	if OneShotBuiltInAgentTypes["general-purpose"] {
		t.Fatal("general-purpose should not be one-shot")
	}
}
