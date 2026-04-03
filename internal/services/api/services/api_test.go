package services

import "testing"

func TestHasTSModule(t *testing.T) {
	if !HasTSModule(EmptyUsage) {
		t.Fatal("emptyUsage expected")
	}
	if HasTSModule("nonexistent.ts") {
		t.Fatal("unknown should be false")
	}
}
