package memdir

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestBuildCombinedMemoryPrompt_shape(t *testing.T) {
	t.Setenv(features.EnvMemorySearchPastContext, "")
	p := BuildCombinedMemoryPrompt(CombinedMemoryPromptOpts{
		AutoMemDir: "/a/memory/",
		TeamMemDir: "/a/memory/team/",
	})
	if !strings.Contains(p, "# Memory") || !strings.Contains(p, "Types of memory") {
		t.Fatal("expected headings")
	}
}
