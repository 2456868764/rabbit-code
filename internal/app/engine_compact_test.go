package app

import (
	"context"
	"testing"

	anthropic "github.com/2456868764/rabbit-code/internal/services/api"

	"github.com/2456868764/rabbit-code/internal/query/engine"
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
