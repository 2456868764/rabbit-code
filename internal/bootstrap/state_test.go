package bootstrap

import (
	"sync"
	"testing"
)

func TestState_concurrent(t *testing.T) {
	t.Parallel()
	s := NewState()
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n * 3)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			s.SetSessionID("sess-a")
			_ = s.SessionID()
		}()
		go func() {
			defer wg.Done()
			s.AddTotalCost(1)
			s.IncMeter()
		}()
		go func() {
			defer wg.Done()
			s.SetCwd("/tmp")
			s.SetProjectRoot("/proj")
			_ = s.Cwd()
			_ = s.ProjectRoot()
		}()
	}
	wg.Wait()
	if s.TotalCost() != uint64(n) {
		t.Fatalf("totalCost: got %d want %d", s.TotalCost(), n)
	}
	if s.MeterEvents() != uint64(n) {
		t.Fatalf("meter: got %d want %d", s.MeterEvents(), n)
	}
}
