package compact

import "strings"

// IsMainThreadPostCompactSource mirrors postCompactCleanup.ts isMainThreadCompact (main-thread-only cache resets).
func IsMainThreadPostCompactSource(querySource string) bool {
	s := strings.TrimSpace(querySource)
	if s == "" {
		return true
	}
	if s == "sdk" {
		return true
	}
	return strings.HasPrefix(s, "repl_main_thread")
}

// ResetMicrocompactStateIfAny calls ResetMicrocompactState when b is non-nil (runPostCompactCleanup first step).
func ResetMicrocompactStateIfAny(b *MicrocompactEditBuffer) {
	if b != nil {
		b.ResetMicrocompactState()
	}
}
