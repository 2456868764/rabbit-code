package anthropic

import (
	"os"
	"path/filepath"
	"testing"
)

func resetDeviceIDCache() {
	deviceIDMu.Lock()
	deviceIDCached = ""
	deviceIDMu.Unlock()
}

func TestLoadOrCreateDeviceID_envWins(t *testing.T) {
	resetDeviceIDCache()
	t.Setenv(EnvRabbitDeviceID, "env-device")
	if got := LoadOrCreateDeviceID(); got != "env-device" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadOrCreateDeviceID_fileStable(t *testing.T) {
	resetDeviceIDCache()
	t.Setenv(EnvRabbitDeviceID, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	// UserConfigDir on darwin uses $HOME/Library/Application Support
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Join(dir, rabbitConfigDirName))

	id1 := LoadOrCreateDeviceID()
	resetDeviceIDCache()
	id2 := LoadOrCreateDeviceID()
	if id1 == "" || id1 != id2 {
		t.Fatalf("id1=%q id2=%q", id1, id2)
	}
	p := filepath.Join(dir, rabbitConfigDirName, deviceIDFileName)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("missing file %s: %v", p, err)
	}
}
