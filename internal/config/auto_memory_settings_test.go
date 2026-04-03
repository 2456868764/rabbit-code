package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTrustedAutoMemoryDirectory_ignoresProject(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	global := t.TempDir()
	user := filepath.Join(global, UserConfigFileName)
	if err := os.WriteFile(user, []byte(`{"autoMemoryDirectory":"/user/mem"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(root, ".rabbit-code.json")
	if err := os.WriteFile(proj, []byte(`{"autoMemoryDirectory":"/evil/mem"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTrustedAutoMemoryDirectory(Paths{
		GlobalConfigDir: global,
		ProjectRoot:     root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/user/mem" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadTrustedAutoMemoryDirectory_policyBeforeUser(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	global := t.TempDir()
	_ = os.WriteFile(filepath.Join(global, UserConfigFileName), []byte(`{"autoMemoryDirectory":"/user/mem"}`), 0o600)
	_ = os.WriteFile(filepath.Join(global, managedSettingsFile), []byte(`{"autoMemoryDirectory":"/policy/mem"}`), 0o600)
	got, err := LoadTrustedAutoMemoryDirectory(Paths{GlobalConfigDir: global, ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/policy/mem" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadTrustedAutoMemoryDirectory_localBeforeUser(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	global := t.TempDir()
	_ = os.WriteFile(filepath.Join(global, UserConfigFileName), []byte(`{"autoMemoryDirectory":"/user/mem"}`), 0o600)
	loc := filepath.Join(root, LocalConfigFileName)
	_ = os.WriteFile(loc, []byte(`{"autoMemoryDirectory":"/local/mem"}`), 0o600)
	got, err := LoadTrustedAutoMemoryDirectory(Paths{GlobalConfigDir: global, ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/local/mem" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadTrustedAutoMemoryDirectory_flagBeforeLocal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	global := t.TempDir()
	_ = os.WriteFile(filepath.Join(global, UserConfigFileName), []byte(`{"autoMemoryDirectory":"/user/mem"}`), 0o600)
	loc := filepath.Join(root, LocalConfigFileName)
	_ = os.WriteFile(loc, []byte(`{"autoMemoryDirectory":"/local/mem"}`), 0o600)
	got, err := LoadTrustedAutoMemoryDirectory(Paths{
		GlobalConfigDir: global,
		ProjectRoot:     root,
		FlagJSON:        `{"autoMemoryDirectory":"/flag/mem"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/flag/mem" {
		t.Fatalf("got %q", got)
	}
}
