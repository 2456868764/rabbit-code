package app

import (
	"context"
	"testing"

	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/query/engine"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestApplyEngineCompactIntegration_bridgesForkPartial(t *testing.T) {
	a := &anthropic.AnthropicAssistant{
		ForkCompactSummary: func(context.Context, []byte, []byte) (string, error) { return "x", nil },
	}
	ApplyEngineCompactIntegration(nil, &engine.Config{}, a)
	if a.ForkPartialCompactSummary == nil {
		t.Fatal("expected ForkPartialCompactSummary after integration")
	}
}

func TestApplyEngineCompactIntegration_defaultAPIContextManagementOpts(t *testing.T) {
	t.Setenv("RABBIT_CODE_ALWAYS_THINKING_DISABLED", "")
	cl := &anthropic.Client{Provider: anthropic.ProviderAnthropic}
	aa := &anthropic.AnthropicAssistant{
		Client:       cl,
		DefaultModel: "claude-sonnet-4-20250514",
	}
	ApplyEngineCompactIntegration(nil, &engine.Config{}, aa)
	if aa.APIContextManagementOpts == nil {
		t.Fatal("expected default APIContextManagementOpts")
	}
	if !aa.APIContextManagementOpts.HasThinking {
		t.Fatalf("want HasThinking for sonnet-4, got %+v", *aa.APIContextManagementOpts)
	}
}

func TestApplyEngineCompactIntegration_respectsExistingAPIContextManagementOpts(t *testing.T) {
	cl := &anthropic.Client{Provider: anthropic.ProviderAnthropic}
	existing := &compact.APIContextManagementOptions{HasThinking: false}
	aa := &anthropic.AnthropicAssistant{
		Client:                   cl,
		DefaultModel:             "claude-sonnet-4-20250514",
		APIContextManagementOpts: existing,
	}
	ApplyEngineCompactIntegration(nil, &engine.Config{Model: "claude-sonnet-4-20250514"}, aa)
	if aa.APIContextManagementOpts != existing || aa.APIContextManagementOpts.HasThinking {
		t.Fatal("expected host opts unchanged")
	}
}
