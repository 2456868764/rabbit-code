package memdir

import "testing"

func TestSessionFragments_nilByDefault(t *testing.T) {
	if s := SessionFragments(); s != nil {
		t.Fatalf("want nil slice, got %#v", s)
	}
}
