package compact

import "sync"

// compactWarningState mirrors compactWarningState.ts compactWarningStore<boolean>(false).
var compactWarningState struct {
	mu   sync.Mutex
	on   bool
	init bool
}

// SuppressCompactWarning mirrors suppressCompactWarning (after successful compaction).
func SuppressCompactWarning() {
	compactWarningState.mu.Lock()
	defer compactWarningState.mu.Unlock()
	compactWarningState.on = true
	compactWarningState.init = true
}

// ClearCompactWarningSuppression mirrors clearCompactWarningSuppression (start of new compact attempt).
func ClearCompactWarningSuppression() {
	compactWarningState.mu.Lock()
	defer compactWarningState.mu.Unlock()
	compactWarningState.on = false
	compactWarningState.init = true
}

// CompactWarningSuppressed reports whether the compact warning should be hidden (useCompactWarningSuppression analogue).
func CompactWarningSuppressed() bool {
	compactWarningState.mu.Lock()
	defer compactWarningState.mu.Unlock()
	if !compactWarningState.init {
		return false
	}
	return compactWarningState.on
}
