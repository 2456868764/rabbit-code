package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot_goMod(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := FindProjectRoot(sub)
	if got != dir {
		t.Fatalf("got %q want %q", got, dir)
	}
}

func TestGlobalConfigDir_xdg(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg-config"))
	t.Setenv("HOME", tmp)
	got, err := GlobalConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "xdg-config", "rabbit-code")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
