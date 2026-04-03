package query

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestProactiveAutoCompactAllowedForQuerySource(t *testing.T) {
	if !ProactiveAutoCompactAllowedForQuerySource("") {
		t.Fatal("main loop empty source should allow")
	}
	if !ProactiveAutoCompactAllowedForQuerySource("main") {
		t.Fatal("arbitrary non-fork source should allow")
	}
	if ProactiveAutoCompactAllowedForQuerySource(QuerySourceSessionMemory) {
		t.Fatal("session_memory should block")
	}
	if ProactiveAutoCompactAllowedForQuerySource(QuerySourceCompact) {
		t.Fatal("compact should block")
	}
	if ProactiveAutoCompactAllowedForQuerySource(QuerySourceExtractMemories) {
		t.Fatal("extract_memories should block")
	}
	t.Run("marble_origami blocked when context collapse on", func(t *testing.T) {
		t.Setenv(features.EnvContextCollapse, "1")
		if ProactiveAutoCompactAllowedForQuerySource(QuerySourceMarbleOrigami) {
			t.Fatal("expected block")
		}
	})
	t.Run("marble_origami allowed when collapse off", func(t *testing.T) {
		t.Setenv(features.EnvContextCollapse, "")
		if !ProactiveAutoCompactAllowedForQuerySource(QuerySourceMarbleOrigami) {
			t.Fatal("expected allow")
		}
	})
}
