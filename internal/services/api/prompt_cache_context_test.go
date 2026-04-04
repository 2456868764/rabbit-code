package anthropic

import (
	"context"
	"testing"
)

func TestPromptCacheBreakContext(t *testing.T) {
	var called bool
	ctx := ContextWithOnPromptCacheBreak(context.Background(), func() { called = true })
	cb, ok := OnPromptCacheBreakFromContext(ctx)
	if !ok || cb == nil {
		t.Fatal()
	}
	cb()
	if !called {
		t.Fatal()
	}
}
