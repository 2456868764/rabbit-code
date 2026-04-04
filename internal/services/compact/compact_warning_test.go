package compact

import (
	"sync/atomic"
	"testing"
)

func TestCompactWarningState_suppressClear(t *testing.T) {
	ClearCompactWarningSuppression()
	if CompactWarningSuppressed() {
		t.Fatal()
	}
	SuppressCompactWarning()
	if !CompactWarningSuppressed() {
		t.Fatal()
	}
	ClearCompactWarningSuppression()
	if CompactWarningSuppressed() {
		t.Fatal()
	}
}

func TestSubscribeCompactWarningSuppression(t *testing.T) {
	ClearCompactWarningSuppression()
	var n atomic.Int32
	unsub := SubscribeCompactWarningSuppression(func() { n.Add(1) })
	SuppressCompactWarning()
	if n.Load() != 1 {
		t.Fatalf("calls=%d", n.Load())
	}
	ClearCompactWarningSuppression()
	if n.Load() != 2 {
		t.Fatalf("calls=%d", n.Load())
	}
	unsub()
	SuppressCompactWarning()
	if n.Load() != 2 {
		t.Fatalf("unsub failed calls=%d", n.Load())
	}
}
