package teamcreatetool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTeamCreate_stubRun(t *testing.T) {
	out, err := New().Run(context.Background(), []byte(`{"team_name":"my-team"}`))
	if err != nil {
		t.Fatal(err)
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.TeamName != "my-team" {
		t.Fatalf("unexpected team_name: %s", o.TeamName)
	}
	if !strings.Contains(o.TeamFilePath, "my-team") {
		t.Fatalf("unexpected team_file_path: %s", o.TeamFilePath)
	}
}

func TestTeamCreate_requiresTeamName(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"team_name":""}`))
	if err == nil {
		t.Fatal("expected error for empty team_name")
	}
}

func TestTeamCreate_rejectsUnknownFields(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"team_name":"x","extra":1}`))
	if err == nil {
		t.Fatal("expected strict JSON error")
	}
}

func TestTeamCreate_delegatesToCallback(t *testing.T) {
	called := false
	rc := &RunContext{
		OnTeamCreate: func(_ context.Context, in Input) (Output, error) {
			called = true
			return Output{
				TeamName:     in.TeamName,
				TeamFilePath: "/real/path/config.json",
				LeadAgentID:  "real-agent-id",
			}, nil
		},
	}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{"team_name":"real-team","description":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("OnTeamCreate not called")
	}
	var o Output
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.LeadAgentID != "real-agent-id" {
		t.Fatalf("unexpected lead_agent_id: %s", o.LeadAgentID)
	}
}

func TestTeamCreateToolName(t *testing.T) {
	if TeamCreateToolName != "TeamCreate" {
		t.Fatalf("unexpected: %q", TeamCreateToolName)
	}
	if New().Name() != TeamCreateToolName {
		t.Fatal("Name() mismatch")
	}
}
