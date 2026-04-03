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
