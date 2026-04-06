package bashtool_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestCheckReadOnlyStructuralConstraints_bareLayout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("ref: refs/heads/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := bashtool.CheckReadOnlyStructuralConstraints("git status", dir)
	if err == nil {
		t.Fatal("expected bare-layout rejection")
	}
}

func TestCheckReadOnlyStructuralConstraints_gitOutsideOriginal(t *testing.T) {
	t.Setenv(features.EnvBashSandboxEnabled, "1")
	t.Setenv(features.EnvBashOriginalWorkdir, "/tmp/orig")
	dir := t.TempDir()
	err := bashtool.CheckReadOnlyStructuralConstraints("git status", dir)
	if err == nil {
		t.Fatal("expected cwd mismatch")
	}
}
