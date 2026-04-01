//go:build linux

package app

import (
	"os"
	"strings"
)

// DetectLibc inspects /proc/self/maps for musl vs glibc (best-effort).
// If maps are unreadable or ambiguous, both values are false.
func DetectLibc() (glibc, musl bool) {
	data, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return false, false
	}
	s := string(data)
	if strings.Contains(s, "libc.musl-") ||
		strings.Contains(s, "ld-musl-") ||
		strings.Contains(s, "/musl/") {
		return false, true
	}
	if strings.Contains(s, "libc.so.6") ||
		strings.Contains(s, "libc-2.") {
		return true, false
	}
	return false, false
}
