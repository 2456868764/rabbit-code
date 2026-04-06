package bashtool_test

import (
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestCheckPathConstraints_processSubstitution(t *testing.T) {
	err := bashtool.CheckPathConstraints(`echo hi > >(tee out)`, bashtool.PathValidationOptions{Cwd: "/"})
	if err == nil {
		t.Fatal("expected rejection")
	}
}

func TestCheckPathConstraints_workdirRoot(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "proj")
	if err := bashtool.CheckPathConstraints(`cd `+sub, bashtool.PathValidationOptions{Cwd: tmp, WorkdirRoot: tmp}); err != nil {
		t.Fatal(err)
	}
	if err := bashtool.CheckPathConstraints(`cd /etc`, bashtool.PathValidationOptions{Cwd: tmp, WorkdirRoot: tmp}); err == nil {
		t.Fatal("expected outside root")
	}
}
