package compact

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestTruncateSessionMemoryForCompact(t *testing.T) {
	longBody := stringsRepeatLine("x", 9000)
	content := "# A\n" + longBody + "\n# B\nshort"
	out, trunc := TruncateSessionMemoryForCompact(content)
	if !trunc {
		t.Fatal("expected truncation")
	}
	if len(out) >= len(content) {
		t.Fatal("expected shorter output")
	}
}

func stringsRepeatLine(prefix string, n int) string {
	var b []byte
	for i := 0; i < n; i++ {
		b = append(b, prefix...)
		b = append(b, '\n')
	}
	return string(b)
}

func TestCalculateSessionMemoryKeepStartIndex_toolPair(t *testing.T) {
	ResetSessionMemoryCompactConfig()
	t.Cleanup(ResetSessionMemoryCompactConfig)
	SetSessionMemoryCompactConfig(SessionMemoryCompactConfig{
		MinTokens:            1,
		MinTextBlockMessages: 1,
		MaxTokens:            1_000_000,
	})
	// user tool_result references tu1; assistant at index 0 has tool_use tu1 — must keep assistant when keeping user tail.
	raw := []byte(`[
		{"role":"assistant","uuid":"a1","content":[{"type":"tool_use","id":"tu1","name":"Read","input":{"file_path":"/x"}}]},
		{"role":"user","uuid":"u1","content":[{"type":"tool_result","tool_use_id":"tu1","content":"ok"}]},
		{"role":"user","uuid":"u2","content":[{"type":"text","text":"tail"}]}
	]`)
	var lines []json.RawMessage
	if err := json.Unmarshal(raw, &lines); err != nil {
		t.Fatal(err)
	}
	idx, err := CalculateSessionMemoryKeepStartIndex(lines, 0)
	if err != nil {
		t.Fatal(err)
	}
	if idx != 0 {
		t.Fatalf("start index %d want 0", idx)
	}
}

func TestTrySessionMemoryCompactionTranscriptJSON(t *testing.T) {
	t.Setenv(features.EnvSessionMemoryFeature, "1")
	t.Setenv(features.EnvSessionMemoryCompactFeature, "1")
	raw := []byte(`[{"role":"user","uuid":"x","content":[{"type":"text","text":"hi"}]}]`)
	hooks := SessionMemoryCompactHooks{
		GetSessionMemoryContent: func(context.Context) (string, error) {
			return "# Mem\nbody", nil
		},
		IsSessionMemoryEmpty: func(context.Context, string) (bool, error) {
			return false, nil
		},
		GetLastSummarizedMessageUUID: func() string { return "x" },
		SessionStartHooks:            func(context.Context, string) ([]json.RawMessage, error) { return nil, nil },
		TranscriptPath:               func() string { return "/t.json" },
	}
	out, ok, err := TrySessionMemoryCompactionTranscriptJSON(context.Background(), raw, "m", "", 0, &hooks)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !json.Valid(out) {
		t.Fatal("invalid json")
	}
	if string(out) == "" {
		t.Fatal()
	}
}

func TestNewSessionMemoryCompactExecutor(t *testing.T) {
	t.Setenv(features.EnvSessionMemoryFeature, "1")
	t.Setenv(features.EnvSessionMemoryCompactFeature, "1")
	ex := NewSessionMemoryCompactExecutor(SessionMemoryCompactHooks{
		GetSessionMemoryContent: func(context.Context) (string, error) {
			return "# M\nx", nil
		},
		IsSessionMemoryEmpty:         func(context.Context, string) (bool, error) { return false, nil },
		GetLastSummarizedMessageUUID: func() string { return "" },
		SessionStartHooks:            func(context.Context, string) ([]json.RawMessage, error) { return nil, nil },
	})
	_, ok, err := ex(context.Background(), "", "m", 0, []byte(`[{"role":"user","uuid":"a","content":[{"type":"text","text":"z"}]}]`))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
