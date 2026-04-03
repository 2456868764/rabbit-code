package query

import "testing"

func TestCloneAutoCompactTracking_nil(t *testing.T) {
	if CloneAutoCompactTracking(nil) != nil {
		t.Fatal()
	}
}

func TestMirrorAutocompactConsecutiveFailures(t *testing.T) {
	st := &LoopState{}
	MirrorAutocompactConsecutiveFailures(st, 2)
	if st.AutoCompactTracking == nil || st.AutoCompactTracking.ConsecutiveFailures == nil || *st.AutoCompactTracking.ConsecutiveFailures != 2 {
		t.Fatalf("got %+v", st.AutoCompactTracking)
	}
	MirrorAutocompactConsecutiveFailures(st, 0)
	if st.AutoCompactTracking == nil || st.AutoCompactTracking.ConsecutiveFailures == nil || *st.AutoCompactTracking.ConsecutiveFailures != 0 {
		t.Fatalf("reset: %+v", st.AutoCompactTracking)
	}
	MirrorAutocompactConsecutiveFailures(nil, 9) // no panic
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
