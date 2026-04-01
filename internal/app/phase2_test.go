package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/bootstrap"
)

func TestApplyManagedEnvFromMerged(t *testing.T) {
	t.Setenv("PHASE2_TEST_VAR", "")
	m := map[string]interface{}{
		"managed_env": map[string]interface{}{
			"PHASE2_TEST_VAR": "from-config",
		},
	}
	keys := ApplyManagedEnvFromMerged(m)
	if len(keys) != 1 || keys[0] != "PHASE2_TEST_VAR" {
		t.Fatalf("keys %v", keys)
	}
	if os.Getenv("PHASE2_TEST_VAR") != "from-config" {
		t.Fatal(os.Getenv("PHASE2_TEST_VAR"))
	}
}

func TestLoadAndApplyMergedConfig(t *testing.T) {
	global := t.TempDir()
	root := t.TempDir()
	_ = os.MkdirAll(global, 0o755)
	path := filepath.Join(global, "config.json")
	content := `{"managed_env":{"PHASE2_TEST_VAR":"x"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PHASE2_TEST_VAR", "")
	st := bootstrap.NewState()
	st.SetProjectRoot(root)
	rt := &Runtime{
		Log:             slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		GlobalConfigDir: global,
		State:           st,
	}
	if err := LoadAndApplyMergedConfig(rt); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PHASE2_TEST_VAR") != "x" {
		t.Fatal(os.Getenv("PHASE2_TEST_VAR"))
	}
}
