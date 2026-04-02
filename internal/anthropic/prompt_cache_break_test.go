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
	if !IsPromptCacheBreakStreamJSON([]byte(`{"error":{"message":"prompt_cache_miss on block"}}`)) {
		t.Fatal("expected prompt_cache_miss heuristic")
	}
	if !IsPromptCacheBreakStreamJSON([]byte(`{"message":"invalid_cache reference"}`)) {
		t.Fatal("expected invalid_cache heuristic")
	}
	if !IsPromptCacheBreakStreamJSON([]byte(`{"type":"error","error":{"message":"cached_prompt invalid"}}`)) {
		t.Fatal("expected cached_prompt+invalid heuristic")
	}
	if !IsPromptCacheBreakStreamJSON([]byte(`cache_key invalid for prompt`)) {
		t.Fatal("expected cache_key+invalid heuristic")
	}
	t.Setenv(features.EnvPromptCacheBreak, "")
	if IsPromptCacheBreakStreamJSON([]byte(`{"type":"error","message":"cache_break"}`)) {
		t.Fatal("expected false when feature off")
	}
}

func TestReadAssistantStream_PromptCacheBreak(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	raw := "data: {\"type\":\"error\",\"error\":{\"message\":\"cache_break\"}}\n\n"
	var called bool
	_, _, err := ReadAssistantStream(context.Background(), strings.NewReader(raw), WithOnPromptCacheBreak(func() { called = true }))
	if !errors.Is(err, ErrPromptCacheBreakDetected) {
		t.Fatalf("got %v", err)
	}
	if !called {
		t.Fatal("expected onPromptCacheBreak")
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
