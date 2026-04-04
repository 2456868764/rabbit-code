package app

import (
	"context"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/query/engine"
)

func TestApplyEngineCompactIntegration_bridgesForkPartial(t *testing.T) {
	a := &query.AnthropicAssistant{
		ForkCompactSummary: func(context.Context, []byte, []byte) (string, error) { return "x", nil },
	}
	ApplyEngineCompactIntegration(nil, &engine.Config{}, a)
	if a.ForkPartialCompactSummary == nil {
		t.Fatal("expected ForkPartialCompactSummary after integration")
	}
}
