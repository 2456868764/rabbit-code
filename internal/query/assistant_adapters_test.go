package query

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestSequenceAssistant_order(t *testing.T) {
	s := &SequenceAssistant{Replies: []string{"a", "b"}}
	v1, err := s.StreamAssistant(context.Background(), "m", 1, nil)
	if err != nil || v1 != "a" {
		t.Fatalf("%q %v", v1, err)
	}
	v2, err := s.StreamAssistant(context.Background(), "m", 1, nil)
	if err != nil || v2 != "b" {
		t.Fatalf("%q %v", v2, err)
	}
	_, err = s.StreamAssistant(context.Background(), "m", 1, nil)
	if !errors.Is(err, ErrSequenceExhausted) {
		t.Fatalf("got %v", err)
	}
}

func TestSequenceTurnAssistant_orderAndTools(t *testing.T) {
	s := &SequenceTurnAssistant{Turns: []TurnResult{
		{Text: "step1", ToolUses: []ToolUseCall{{ID: "tu1", Name: "bash", Input: json.RawMessage(`{"x":1}`)}}},
		{Text: "done"},
	}}
	r1, err := s.AssistantTurn(context.Background(), "m", 1, nil)
	if err != nil || r1.Text != "step1" || len(r1.ToolUses) != 1 || r1.ToolUses[0].ID != "tu1" {
		t.Fatalf("%+v %v", r1, err)
	}
	r2, err := s.AssistantTurn(context.Background(), "m", 1, nil)
	if err != nil || r2.Text != "done" || len(r2.ToolUses) != 0 {
		t.Fatalf("%+v %v", r2, err)
	}
	_, err = s.AssistantTurn(context.Background(), "m", 1, nil)
	if !errors.Is(err, ErrSequenceExhausted) {
		t.Fatalf("got %v", err)
	}
}

func TestStreamAsTurnAssistant_noTools(t *testing.T) {
	a := StreamAsTurnAssistant(StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
		return "hi", nil
	}))
	r, err := a.AssistantTurn(context.Background(), "m", 1, nil)
	if err != nil || r.Text != "hi" || len(r.ToolUses) != 0 {
		t.Fatalf("%+v %v", r, err)
	}
}
