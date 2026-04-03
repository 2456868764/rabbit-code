package app

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/bootstrap"
	"github.com/2456868764/rabbit-code/internal/config"
	"github.com/2456868764/rabbit-code/internal/query/engine"
)

func TestApplyEngineMemdirFromRuntime(t *testing.T) {
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
	rt := &Runtime{
		Log:             slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		GlobalConfigDir: global,
		State:           bootstrap.NewState(),
		MergedSettings:  map[string]interface{}{"autoMemoryEnabled": true, "x": float64(1)},
	}
	rt.State.SetProjectRoot(root)
	var cfg engine.Config
	if err := ApplyEngineMemdirFromRuntime(rt, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.MemdirTrustedAutoMemoryDirectory != mem {
		t.Fatalf("trusted %q", cfg.MemdirTrustedAutoMemoryDirectory)
	}
	if cfg.InitialSettings == nil || cfg.InitialSettings["x"].(float64) != 1 {
		t.Fatalf("initial %+v", cfg.InitialSettings)
	}
}
