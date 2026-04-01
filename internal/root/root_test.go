package root

import "testing"

func TestOK(t *testing.T) {
	if !OK() {
		t.Fatal("OK() = false")
	}
}
