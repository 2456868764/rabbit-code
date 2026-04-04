package compact

import "testing"

func TestCompactWarningState_suppressClear(t *testing.T) {
	ClearCompactWarningSuppression()
	if CompactWarningSuppressed() {
		t.Fatal()
	}
	SuppressCompactWarning()
	if !CompactWarningSuppressed() {
		t.Fatal()
	}
	ClearCompactWarningSuppression()
	if CompactWarningSuppressed() {
		t.Fatal()
	}
}
