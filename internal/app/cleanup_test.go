package app

import (
	"testing"
)

func TestCleanupRegistry_LIFO(t *testing.T) {
	t.Parallel()
	var order []int
	r := &CleanupRegistry{}
	r.Register(func() { order = append(order, 1) })
	r.Register(func() { order = append(order, 2) })
	r.Register(func() { order = append(order, 3) })
	r.Run()
	if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Fatalf("order %v want [3 2 1]", order)
	}
	r.Run() // idempotent
}
