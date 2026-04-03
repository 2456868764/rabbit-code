package query

import "testing"

func TestCloneAutoCompactTracking_nil(t *testing.T) {
	if CloneAutoCompactTracking(nil) != nil {
		t.Fatal()
	}
}

func TestCloneAutoCompactTracking_copiesPointerField(t *testing.T) {
	v := 3
	orig := &AutoCompactTracking{Compacted: true, TurnCounter: 2, TurnID: "t1", ConsecutiveFailures: &v}
	cp := CloneAutoCompactTracking(orig)
	if cp == nil || cp == orig {
		t.Fatal()
	}
	if *cp.ConsecutiveFailures != 3 {
		t.Fatal()
	}
	*cp.ConsecutiveFailures = 99
	if *orig.ConsecutiveFailures != 3 {
		t.Fatal("mutating clone should not affect original")
	}
}
