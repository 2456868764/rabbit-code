package compact

import (
	"os"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestMicroCompactRequested(t *testing.T) {
	_ = os.Unsetenv("RABBIT_CODE_CACHED_MICROCOMPACT")
	if MicroCompactRequested() {
		t.Fatal()
	}
	t.Setenv("RABBIT_CODE_CACHED_MICROCOMPACT", "1")
	if !MicroCompactRequested() {
		t.Fatal()
	}
}

func TestPromptCacheBreakActive(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	if !PromptCacheBreakActive() {
		t.Fatal()
	}
}
