package features

import (
	"testing"
)

func TestPhase5Env_defaultsOff(t *testing.T) {
	t.Setenv(EnvTokenBudget, "")
	t.Setenv(EnvReactiveCompact, "")
	if TokenBudgetEnabled() || ReactiveCompactEnabled() {
		t.Fatal("expected off")
	}
	if TokenBudgetMaxInputBytes() != 0 {
		t.Fatalf("got %d", TokenBudgetMaxInputBytes())
	}
}

func TestTokenBudgetMaxInputBytes_whenEnabled(t *testing.T) {
	t.Setenv(EnvTokenBudget, "1")
	t.Setenv(EnvTokenBudgetMaxInputBytes, "")
	if TokenBudgetMaxInputBytes() != 4_000_000 {
		t.Fatalf("default %d", TokenBudgetMaxInputBytes())
	}
	t.Setenv(EnvTokenBudgetMaxInputBytes, "100")
	if TokenBudgetMaxInputBytes() != 100 {
		t.Fatalf("got %d", TokenBudgetMaxInputBytes())
	}
}

func TestSnipCompactEnabled(t *testing.T) {
	t.Setenv(EnvSnipCompact, "true")
	if !SnipCompactEnabled() {
		t.Fatal()
	}
}
