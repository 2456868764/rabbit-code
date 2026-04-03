package query

import "testing"

func TestMarshalAutoCompactTrackingJSON_roundTrip(t *testing.T) {
	v := 2
	orig := &AutoCompactTracking{Compacted: true, TurnCounter: 3, TurnID: "autocompact:1", ConsecutiveFailures: &v}
	data, err := MarshalAutoCompactTrackingJSON(orig)
	if err != nil {
		t.Fatal(err)
	}
	out, err := UnmarshalAutoCompactTrackingJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || !out.Compacted || out.TurnCounter != 3 || out.TurnID != "autocompact:1" ||
		out.ConsecutiveFailures == nil || *out.ConsecutiveFailures != 2 {
		t.Fatalf("%+v", out)
	}
	if _, err := UnmarshalAutoCompactTrackingJSON([]byte("  ")); err != nil {
		t.Fatal(err)
	}
}

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
