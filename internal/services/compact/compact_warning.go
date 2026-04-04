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
	notifyCompactWarningSubscribers()
}

// ClearCompactWarningSuppression mirrors clearCompactWarningSuppression (start of new compact attempt).
func ClearCompactWarningSuppression() {
	compactWarningState.mu.Lock()
	defer compactWarningState.mu.Unlock()
	compactWarningState.on = false
	compactWarningState.init = true
	notifyCompactWarningSubscribers()
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

// compactWarningSubscribers mirrors compactWarningHook.ts useSyncExternalStore subscription side.
var compactWarningSubscribers struct {
	mu   sync.Mutex
	next int
	subs map[int]func()
}

// SubscribeCompactWarningSuppression registers fn to run whenever suppression toggles (suppress or clear).
// Returns unsubscribe closure. Snapshot value is CompactWarningSuppressed().
func SubscribeCompactWarningSuppression(fn func()) (unsubscribe func()) {
	if fn == nil {
		return func() {}
	}
	compactWarningSubscribers.mu.Lock()
	if compactWarningSubscribers.subs == nil {
		compactWarningSubscribers.subs = make(map[int]func())
	}
	compactWarningSubscribers.next++
	id := compactWarningSubscribers.next
	compactWarningSubscribers.subs[id] = fn
	compactWarningSubscribers.mu.Unlock()
	return func() {
		compactWarningSubscribers.mu.Lock()
		delete(compactWarningSubscribers.subs, id)
		compactWarningSubscribers.mu.Unlock()
	}
}

func notifyCompactWarningSubscribers() {
	compactWarningSubscribers.mu.Lock()
	subs := make([]func(), 0, len(compactWarningSubscribers.subs))
	for _, fn := range compactWarningSubscribers.subs {
		subs = append(subs, fn)
	}
	compactWarningSubscribers.mu.Unlock()
	for _, fn := range subs {
		fn()
	}
}
