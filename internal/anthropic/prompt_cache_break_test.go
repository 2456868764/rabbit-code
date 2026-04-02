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
	if !IsPromptCacheBreakStreamJSON([]byte(`{"error":{"type":"invalid_request_error","message":"prompt_cache_invalid"}}`)) {
		t.Fatal("expected prompt_cache_invalid heuristic")
	}
	if !IsPromptCacheBreakStreamJSON([]byte(`{"message":"invalidated_prompt cache key"}`)) {
		t.Fatal("expected invalidated_prompt heuristic")
	}
	if !IsPromptCacheBreakStreamJSON([]byte(`{"detail":"prompt_cache expired for this session"}`)) {
		t.Fatal("expected prompt_cache+expired heuristic")
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
