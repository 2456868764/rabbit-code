package processuserinput

import "testing"

func TestMatchesNegativeKeyword(t *testing.T) {
	if MatchesNegativeKeyword("hello") {
		t.Fatal("plain")
	}
	if !MatchesNegativeKeyword("this sucks") {
		t.Fatal("this sucks")
	}
	if !MatchesNegativeKeyword("What the hell") {
		t.Fatal("what the hell")
	}
	if MatchesNegativeKeyword("ship it") {
		t.Fatal("false positive shit substring in ship")
	}
}

func TestMatchesKeepGoingKeyword(t *testing.T) {
	if !MatchesKeepGoingKeyword("continue") {
		t.Fatal("exact continue")
	}
	if MatchesKeepGoingKeyword("  continue please  ") {
		t.Fatal("continue not whole prompt")
	}
	if !MatchesKeepGoingKeyword("please keep going") {
		t.Fatal("keep going")
	}
	if !MatchesKeepGoingKeyword("go on with the plan") {
		t.Fatal("go on")
	}
}

func TestPlainPromptSignals(t *testing.T) {
	n, k := PlainPromptSignals("keep going")
	if n || !k {
		t.Fatalf("got n=%v k=%v", n, k)
	}
}
