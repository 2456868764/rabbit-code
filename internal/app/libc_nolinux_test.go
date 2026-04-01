//go:build !linux

package app

import "testing"

func TestDetectLibc_nonLinux(t *testing.T) {
	g, m := DetectLibc()
	if g || m {
		t.Fatalf("expected false,false on non-linux, got glibc=%v musl=%v", g, m)
	}
}
