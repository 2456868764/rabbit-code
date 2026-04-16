package teamdeletetool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTeamDelete_stubRun(t *testing.T) {
	out, err := New().Run(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if !o.Success {
		t.Fatalf("stub should return success: %v", o)
	}
}

func TestTeamDelete_rejectsUnknownFields(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"extra":1}`))
	if err == nil {
		t.Fatal("expected strict JSON error")
	}
}

func TestTeamDelete_delegatesToCallback(t *testing.T) {
	called := false
	rc := &RunContext{
		TeamName: "my-team",
		OnTeamDelete: func(_ context.Context, teamName string) (Output, error) {
			called = true
			return Output{Success: true, Message: "cleaned up", TeamName: teamName}, nil
		},
	}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("OnTeamDelete not called")
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.TeamName != "my-team" {
		t.Fatalf("unexpected team_name: %s", o.TeamName)
	}
}

func TestTeamDeleteToolName(t *testing.T) {
	if TeamDeleteToolName != "TeamDelete" {
		t.Fatalf("unexpected: %q", TeamDeleteToolName)
	}
	if New().Name() != TeamDeleteToolName {
		t.Fatal("Name() mismatch")
	}
}
