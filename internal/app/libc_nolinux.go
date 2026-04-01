//go:build !linux

package app

// DetectLibc returns glibc/musl hints. Non-Linux builds have no dynamic libc classification here.
func DetectLibc() (glibc, musl bool) {
	return false, false
}
