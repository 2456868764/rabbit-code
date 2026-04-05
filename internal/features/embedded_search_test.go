package features

import (
	"testing"
)

func TestHasEmbeddedSearchTools(t *testing.T) {
	t.Setenv(EnvEmbeddedSearchTools, "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("RABBIT_CODE_ENTRYPOINT", "")
	if HasEmbeddedSearchTools() {
		t.Fatal("unset off")
	}
	t.Setenv(EnvEmbeddedSearchTools, "1")
	if !HasEmbeddedSearchTools() {
		t.Fatal("on")
	}
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-ts")
	if HasEmbeddedSearchTools() {
		t.Fatal("sdk-ts off")
	}
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("RABBIT_CODE_ENTRYPOINT", "sdk-cli")
	if HasEmbeddedSearchTools() {
		t.Fatal("sdk-cli off")
	}
}

func TestReplModeEnabled(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REPL", "")
	t.Setenv("RABBIT_CODE_REPL", "")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("RABBIT_CODE_REPL_MODE", "")
	if ReplModeEnabled() {
		t.Fatal("default off")
	}
	t.Setenv("CLAUDE_REPL_MODE", "1")
	if !ReplModeEnabled() {
		t.Fatal("legacy on")
	}
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	if ReplModeEnabled() {
		t.Fatal("explicit repl off")
	}
}

func TestUseShellGrepForMemoryPrompts(t *testing.T) {
	t.Setenv(EnvEmbeddedSearchTools, "")
	t.Setenv("CLAUDE_CODE_REPL", "")
	t.Setenv("RABBIT_CODE_REPL", "")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("RABBIT_CODE_REPL_MODE", "")
	if UseShellGrepForMemoryPrompts() {
		t.Fatal("default off")
	}
	t.Setenv(EnvEmbeddedSearchTools, "1")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("RABBIT_CODE_ENTRYPOINT", "")
	if !UseShellGrepForMemoryPrompts() {
		t.Fatal("embedded on")
	}
	t.Setenv(EnvEmbeddedSearchTools, "")
	t.Setenv("RABBIT_CODE_REPL_MODE", "true")
	if !UseShellGrepForMemoryPrompts() {
		t.Fatal("repl on")
	}
}
