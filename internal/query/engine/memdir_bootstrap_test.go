package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/config"
)

func TestApplyMergedSettingsForMemdir(t *testing.T) {
	global := t.TempDir()
	root := t.TempDir()
	mem := t.TempDir()
	user := filepath.Join(global, config.UserConfigFileName)
	payload, err := json.Marshal(map[string]string{"autoMemoryDirectory": mem})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(user, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	merged := map[string]interface{}{"autoMemoryEnabled": false, "k": "v"}
	var cfg Config
	if err := ApplyMergedSettingsForMemdir(&cfg, config.Paths{
		GlobalConfigDir: global,
		ProjectRoot:     root,
	}, merged); err != nil {
		t.Fatal(err)
	}
	if cfg.MemdirTrustedAutoMemoryDirectory != mem {
		t.Fatalf("trusted %q", cfg.MemdirTrustedAutoMemoryDirectory)
	}
	if cfg.InitialSettings == nil || cfg.InitialSettings["k"] != "v" {
		t.Fatalf("initial %+v", cfg.InitialSettings)
	}
	merged["k"] = "mut"
	if cfg.InitialSettings["k"] != "v" {
		t.Fatal("expected shallow clone of map shell")
	}
}
