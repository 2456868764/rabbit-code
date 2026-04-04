package query

import "testing"

func TestBuildQueryConfig_defaults(t *testing.T) {
	t.Setenv(envStreamingToolExecutionRabbit, "")
	t.Setenv(envStreamingToolExecutionClaude, "")
	t.Setenv(envEmitToolUseSummariesRabbit, "")
	t.Setenv(envEmitToolUseSummariesClaude, "")
	t.Setenv(envUserTypeRabbit, "")
	t.Setenv(envUserType, "")
	t.Setenv(envDisableFastModeRabbit, "")
	t.Setenv(envDisableFastModeClaude, "")

	c := BuildQueryConfig("sess-1")
	if c.SessionID != "sess-1" {
		t.Fatalf("session %q", c.SessionID)
	}
	if !c.Gates.StreamingToolExecution {
		t.Fatal("expected streaming true by default")
	}
	if c.Gates.EmitToolUseSummaries {
		t.Fatal("emit summaries off when env unset")
	}
	if c.Gates.IsAnt {
		t.Fatal("isAnt false when USER_TYPE unset")
	}
	if !c.Gates.FastModeEnabled {
		t.Fatal("fast mode on when DISABLE unset")
	}
}

func TestBuildQueryConfig_emitAndFast(t *testing.T) {
	t.Setenv(envEmitToolUseSummariesClaude, "1")
	t.Setenv(envDisableFastModeClaude, "true")
	c := BuildQueryConfig("")
	if !c.Gates.EmitToolUseSummaries {
		t.Fatal("emit on")
	}
	if c.Gates.FastModeEnabled {
		t.Fatal("fast off when disable truthy")
	}
}

func TestBuildQueryConfig_isAnt(t *testing.T) {
	t.Setenv(envUserType, "ant")
	c := BuildQueryConfig("")
	if !c.Gates.IsAnt {
		t.Fatal("isAnt")
	}
}
