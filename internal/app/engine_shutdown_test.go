package app

import (
	"context"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/engine"
)

func TestRuntime_RegisterEngineShutdown_nilEngine(t *testing.T) {
	rt := &Runtime{Cleanup: &CleanupRegistry{}}
	rt.RegisterEngineShutdown(nil)
	rt.Close()
}

func TestRuntime_RegisterEngineShutdown_orderBeforeEarlierCleanups(t *testing.T) {
	rt := &Runtime{Cleanup: &CleanupRegistry{}}
	var seq []int
	rt.Cleanup.Register(func() { seq = append(seq, 2) })
	e := engine.New(context.Background(), nil)
	rt.RegisterEngineShutdown(e)
	rt.Cleanup.Register(func() { seq = append(seq, 1) })
	rt.Close()
	// LIFO: last registered runs first — 1, then engine drain, then 2.
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
