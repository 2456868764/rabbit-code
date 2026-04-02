package querydeps

import (
	"context"
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
