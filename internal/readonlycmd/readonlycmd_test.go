package readonlycmd

import "testing"

func TestIsCommandSafeViaFlagParsing_gitDiffStat(t *testing.T) {
	raw := "git diff --stat"
	toks, err := TokenizeShellWords(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !IsCommandSafeViaFlagParsing(raw, toks) {
		t.Fatal("expected git diff --stat allowed")
	}
}

func TestIsCommandSafeViaFlagParsing_gitShowShorthand(t *testing.T) {
	raw := "git show -1 --name-only"
	toks, err := TokenizeShellWords(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !IsCommandSafeViaFlagParsing(raw, toks) {
		t.Fatal("expected git show with numeric shorthand allowed")
	}
}
