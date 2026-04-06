package bashtool

import "testing"

func TestGetDestructiveCommandWarning_gitClean(t *testing.T) {
	if GetDestructiveCommandWarning("git clean -fd") == "" {
		t.Fatal("expect warning for git clean -f")
	}
	if GetDestructiveCommandWarning("git clean -n") != "" {
		t.Fatal("dry-run -n: no warning")
	}
	if GetDestructiveCommandWarning("git clean --dry-run -f") != "" {
		t.Fatal("--dry-run: no warning")
	}
	if GetDestructiveCommandWarning("git clean -x") != "" {
		t.Fatal("no -f: no warning")
	}
}
