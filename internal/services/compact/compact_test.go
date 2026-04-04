package compact

import (
	"errors"
	"testing"
)

func TestRunPhase_String(t *testing.T) {
	if g, w := RunIdle.String(), "idle"; g != w {
		t.Fatalf("%q", g)
	}
}

func TestRunPhase_Next(t *testing.T) {
	if p := RunIdle.Next(true, false); p != RunAutoPending {
		t.Fatal(p)
	}
	if p := RunIdle.Next(false, true); p != RunReactivePending {
		t.Fatal(p)
	}
	if p := RunAutoPending.Next(false, false); p != RunExecuting {
		t.Fatal(p)
	}
	if p := RunExecuting.Next(false, false); p != RunIdle {
		t.Fatal(p)
	}
}

func TestParsePhase(t *testing.T) {
	if ParsePhase("auto_pending") != RunAutoPending {
		t.Fatal()
	}
	if ParsePhase("") != RunIdle {
		t.Fatal()
	}
}

func TestAfterSuccessfulCompactExecution(t *testing.T) {
	if g, w := AfterSuccessfulCompactExecution(RunExecuting), RunIdle; g != w {
		t.Fatalf("executing -> idle: got %v want %v", g, w)
	}
	if g, w := AfterSuccessfulCompactExecution(RunReactivePending), RunReactivePending; g != w {
		t.Fatalf("pending unchanged: got %v want %v", g, w)
	}
}

func TestExecutorPhaseAfterSchedule(t *testing.T) {
	if g, w := ExecutorPhaseAfterSchedule(RunAutoPending), RunExecuting; g != w {
		t.Fatalf("auto_pending: got %v want %v", g, w)
	}
	if g, w := ExecutorPhaseAfterSchedule(RunReactivePending), RunExecuting; g != w {
		t.Fatalf("reactive_pending: got %v want %v", g, w)
	}
	if g, w := ExecutorPhaseAfterSchedule(RunIdle), RunIdle; g != w {
		t.Fatalf("idle: got %v want %v", g, w)
	}
}

func TestResultPhaseAfterCompactExecutor(t *testing.T) {
	if g, w := ResultPhaseAfterCompactExecutor(RunExecuting, nil), RunIdle; g != w {
		t.Fatalf("success: got %v want %v", g, w)
	}
	if g, w := ResultPhaseAfterCompactExecutor(RunExecuting, errors.New("fail")), RunExecuting; g != w {
		t.Fatalf("error: got %v want %v", g, w)
	}
}

// Expected names match restored-src/src/services/compact/microCompact.ts COMPACTABLE_TOOLS
// (FILE_READ_TOOL_NAME, SHELL_TOOL_NAMES, GREP, GLOB, WEB_SEARCH, WEB_FETCH, FILE_EDIT, FILE_WRITE).
func TestCompactableToolNames_matchMicroCompactTS(t *testing.T) {
	want := []string{
		"Read", "Bash", "PowerShell", "Grep", "Glob",
		"WebSearch", "WebFetch", "Edit", "Write",
	}
	for _, name := range want {
		if !IsCompactableToolName(name) {
			t.Errorf("expected %q to be compactable (drift vs microCompact.ts COMPACTABLE_TOOLS?)", name)
		}
	}
	if IsCompactableToolName("TodoWrite") {
		t.Fatal("TodoWrite should not be compactable")
	}
}
