package compact

import "testing"

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
