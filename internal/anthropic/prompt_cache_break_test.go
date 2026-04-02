package anthropic

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestIsPromptCacheBreakStreamJSON(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	if !IsPromptCacheBreakStreamJSON([]byte(`{"type":"error","message":"cache_break"}`)) {
		t.Fatal()
	}
	t.Setenv(features.EnvPromptCacheBreak, "")
	if IsPromptCacheBreakStreamJSON([]byte(`{"type":"error","message":"cache_break"}`)) {
		t.Fatal("expected false when feature off")
	}
}

func TestReadAssistantStream_PromptCacheBreak(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	raw := "data: {\"type\":\"error\",\"error\":{\"message\":\"cache_break\"}}\n\n"
	_, _, err := ReadAssistantStream(context.Background(), strings.NewReader(raw))
	if !errors.Is(err, ErrPromptCacheBreakDetected) {
		t.Fatalf("got %v", err)
	}
}

func TestReadAssistantStream_ErrorWhenBreakDetectionOff(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "")
	raw := "data: {\"type\":\"error\",\"error\":{\"message\":\"cache_break\"}}\n\n"
	_, _, err := ReadAssistantStream(context.Background(), strings.NewReader(raw))
	if err == nil || errors.Is(err, ErrPromptCacheBreakDetected) {
		t.Fatalf("got %v", err)
	}
}
