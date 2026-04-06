package bashtool_test

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestSedCommandAllowedByAllowlist_linePrint(t *testing.T) {
	if !bashtool.SedCommandAllowedByAllowlist(`sed -n '1p'`) {
		t.Fatal("line print stdin")
	}
	if !bashtool.SedCommandAllowedByAllowlist(`sed -n '1p' README.md`) {
		t.Fatal("line print with file")
	}
	if bashtool.SedCommandAllowedByAllowlist(`sed '1p'`) {
		t.Fatal("missing -n")
	}
}

func TestSedCommandAllowedByAllowlist_substitution(t *testing.T) {
	if !bashtool.SedCommandAllowedByAllowlist(`sed 's/a/b/'`) {
		t.Fatal("stdout substitution")
	}
	if bashtool.SedCommandAllowedByAllowlist(`sed 's/a/b/' f.txt`) {
		t.Fatal("substitution with file disallowed without path layer")
	}
}

func TestSedCommandAllowedByAllowlist_dangerous(t *testing.T) {
	if bashtool.SedCommandAllowedByAllowlist(`sed -n '1w/tmp/x'`) {
		t.Fatal("write command")
	}
}

func TestParseSedEditCommand(t *testing.T) {
	info := bashtool.ParseSedEditCommand(`sed -i '' 's/foo/bar/g' ./x.txt`)
	if info == nil {
		t.Fatal("expected parse")
	}
	if info.Pattern != "foo" || info.Replacement != "bar" || info.Flags != "g" {
		t.Fatalf("%+v", info)
	}
}

func TestValidatePipePermissionPreflight_cdAndGitCompound(t *testing.T) {
	err := bashtool.ValidatePipePermissionPreflight("cd /tmp && git status")
	if err == nil {
		t.Fatal("expected cd+git rejection")
	}
}
