package app

import (
	"context"
	"testing"
)

func TestWireHeadlessEngineForShutdown_nilRuntime(t *testing.T) {
	WireHeadlessEngineForShutdown(nil, context.Background())
}

func TestWireHeadlessEngineForShutdown_cleanupOrder(t *testing.T) {
	rt := &Runtime{Cleanup: &CleanupRegistry{}}
	var seq []int
	rt.Cleanup.Register(func() { seq = append(seq, 2) })
	WireHeadlessEngineForShutdown(rt, context.Background())
	rt.Cleanup.Register(func() { seq = append(seq, 1) })
	rt.Close()
	want := []int{1, 2}
	if len(seq) != len(want) {
		t.Fatalf("seq=%v want %v", seq, want)
	}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("seq=%v want %v", seq, want)
		}
	}
}
