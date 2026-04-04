package query

import (
	"context"
	"fmt"
	"testing"

	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestStreamingCompactExecutor_nilClient(t *testing.T) {
	ex := StreamingCompactExecutor(nil, "")
	_, _, err := ex(context.Background(), compact.RunExecuting, []byte(`[]`))
	if err != anthropic.ErrNilAnthropicClient {
		t.Fatalf("got %v", err)
	}
}

func TestStreamAssistantFunc(t *testing.T) {
	var f StreamAssistantFunc = func(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
		return fmt.Sprintf("%s:%d:%s", model, maxTokens, string(messagesJSON)), nil
	}
	out, err := f.StreamAssistant(context.Background(), "x", 3, []byte(`[]`))
	if err != nil || out != "x:3:[]" {
		t.Fatalf("%q %v", out, err)
	}
}
