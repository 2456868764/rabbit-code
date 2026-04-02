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

func TestTokenBudgetMaxInputTokens_whenEnabled(t *testing.T) {
	t.Setenv(EnvTokenBudget, "1")
	if TokenBudgetMaxInputTokens() != 0 {
		t.Fatalf("unset want 0 got %d", TokenBudgetMaxInputTokens())
	}
	t.Setenv(EnvTokenBudgetMaxInputTokens, "500")
	if TokenBudgetMaxInputTokens() != 500 {
		t.Fatalf("got %d", TokenBudgetMaxInputTokens())
	}
}

func TestTokenBudgetMaxAttachmentBytes_whenEnabled(t *testing.T) {
	t.Setenv(EnvTokenBudget, "1")
	if TokenBudgetMaxAttachmentBytes() != 0 {
		t.Fatal()
	}
	t.Setenv(EnvTokenBudgetMaxAttachmentBytes, "99")
	if TokenBudgetMaxAttachmentBytes() != 99 {
		t.Fatal()
	}
}

func TestSessionRestoreEnabled(t *testing.T) {
	t.Setenv(EnvSessionRestore, "true")
	if !SessionRestoreEnabled() {
		t.Fatal()
	}
}

func TestSnipCompactEnabled(t *testing.T) {
	t.Setenv(EnvSnipCompact, "true")
	if !SnipCompactEnabled() {
		t.Fatal()
	}
}

func TestReactiveCompactMinTranscriptBytes(t *testing.T) {
	t.Setenv(EnvReactiveCompact, "1")
	t.Setenv(EnvReactiveCompactMinBytes, "50")
	if ReactiveCompactMinTranscriptBytes() != 50 {
		t.Fatal()
	}
}

func TestReactiveCompactMinEstimatedTokens(t *testing.T) {
	t.Setenv(EnvReactiveCompact, "1")
	if ReactiveCompactMinEstimatedTokens() != 0 {
		t.Fatal()
	}
	t.Setenv(EnvReactiveCompactMinTokens, "100")
	if ReactiveCompactMinEstimatedTokens() != 100 {
		t.Fatal()
	}
}

func TestHistorySnipThresholds(t *testing.T) {
	t.Setenv(EnvHistorySnip, "true")
	t.Setenv(EnvHistorySnipMaxBytes, "99")
	if HistorySnipMaxBytes() != 99 {
		t.Fatal()
	}
}

func TestSnipCompactThresholds(t *testing.T) {
	t.Setenv(EnvSnipCompact, "true")
	t.Setenv(EnvSnipCompactMaxBytes, "100")
	if SnipCompactMaxBytes() != 100 {
		t.Fatal()
	}
	t.Setenv(EnvHistorySnip, "")
	if HistorySnipMaxBytes() != 0 {
		t.Fatal("history snip should be off")
	}
}

func TestTemplateMarkdownDir(t *testing.T) {
	t.Setenv(EnvTemplates, "1")
	t.Setenv(EnvTemplateDir, "/tmp/tpl")
	if TemplateMarkdownDir() != "/tmp/tpl" {
		t.Fatal()
	}
}

func TestTemplateNames(t *testing.T) {
	t.Setenv(EnvTemplates, "true")
	t.Setenv(EnvTemplateNames, " a , b ")
	n := TemplateNames()
	if len(n) != 2 || n[0] != "a" || n[1] != "b" {
		t.Fatalf("%#v", n)
	}
}
