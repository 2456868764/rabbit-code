package thinking

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestInterleavedAPIContextManagementOpts_sonnet4_vs_haiku3(t *testing.T) {
	t.Setenv(features.EnvRedactThinking, "")
	t.Setenv(features.EnvThinkingClearAll, "")
	t.Setenv("RABBIT_CODE_ALWAYS_THINKING_DISABLED", "")

	sonnet := InterleavedAPIContextManagementOpts("claude-sonnet-4-20250514", ProviderAnthropic)
	if !sonnet.HasThinking {
		t.Fatalf("sonnet-4 anthropic: want HasThinking true, got %+v", sonnet)
	}
	if sonnet.IsRedactThinkingActive || sonnet.ClearAllThinking {
		t.Fatalf("unexpected flags without env: %+v", sonnet)
	}

	haiku := InterleavedAPIContextManagementOpts("claude-3-5-haiku-20241022", ProviderAnthropic)
	if haiku.HasThinking {
		t.Fatalf("haiku 3.x: want HasThinking false, got %+v", haiku)
	}
}

func TestInterleavedAPIContextManagementOpts_envRedactAndClearAll(t *testing.T) {
	t.Setenv(features.EnvRedactThinking, "1")
	t.Setenv(features.EnvThinkingClearAll, "true")

	o := InterleavedAPIContextManagementOpts("claude-sonnet-4-20250514", ProviderAnthropic)
	if !o.IsRedactThinkingActive || !o.ClearAllThinking {
		t.Fatalf("want redact + clear-all from env: %+v", o)
	}
}
