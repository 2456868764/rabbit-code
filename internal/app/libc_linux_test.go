//go:build linux

package app

import "testing"

func TestDetectLibc_linux_selfConsistent(t *testing.T) {
	g, m := DetectLibc()
	if g && m {
		t.Fatal("glibc and musl should be mutually exclusive")
	}
	// Typical CI images are either glibc or musl; at least one may be true.
	t.Logf("DetectLibc: glibc=%v musl=%v", g, m)
}
