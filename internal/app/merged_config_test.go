package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/bootstrap"
)

const testManagedEnvKey = "RABBIT_CODE_MANAGED_ENV_TEST_KEY"

func TestApplyManagedEnvFromMerged(t *testing.T) {
	t.Setenv(testManagedEnvKey, "")
	m := map[string]interface{}{
		"managed_env": map[string]interface{}{
			testManagedEnvKey: "from-config",
		},
	}
	keys := ApplyManagedEnvFromMerged(m)
	if len(keys) != 1 || keys[0] != testManagedEnvKey {
		t.Fatalf("keys %v", keys)
	}
	if os.Getenv(testManagedEnvKey) != "from-config" {
		t.Fatal(os.Getenv(testManagedEnvKey))
	}
}

func TestLoadAndApplyMergedConfig(t *testing.T) {
	global := t.TempDir()
	root := t.TempDir()
	_ = os.MkdirAll(global, 0o755)
	path := filepath.Join(global, "config.json")
	content := `{"managed_env":{"` + testManagedEnvKey + `":"x"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(testManagedEnvKey, "")
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
	if os.Getenv(testManagedEnvKey) != "x" {
		t.Fatal(os.Getenv(testManagedEnvKey))
	}
}
