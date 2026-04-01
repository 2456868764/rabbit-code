package app

import (
	"fmt"
	"os"

	"github.com/2456868764/rabbit-code/internal/config"
)

// LoadAndApplyMergedConfig runs after trust: LoadMerged + ApplyManagedEnvFromMerged (Phase 2).
func LoadAndApplyMergedConfig(rt *Runtime) error {
	if rt == nil {
		return nil
	}
	root := ""
	if rt.State != nil {
		root = rt.State.ProjectRoot()
	}
	m, err := config.LoadMerged(config.Paths{
		GlobalConfigDir: rt.GlobalConfigDir,
		ProjectRoot:     root,
	})
	if err != nil {
		return err
	}
	cwd, _ := os.Getwd()
	extraPaths := config.ExtraCAPEMPaths(m, root, cwd)
	if len(extraPaths) > 0 && rt.RootCAs != nil {
		if err := AppendPEMFiles(rt.RootCAs, extraPaths); err != nil {
			return fmt.Errorf("extra_ca_paths: %w", err)
		}
	}
	keys := ApplyManagedEnvFromMerged(m)
	if rt.Log != nil && len(keys) > 0 {
		rt.Log.Debug("managed_env applied", "keys", keys)
	}
	return nil
}

// RunConfigDump loads merged settings (no trust required) and prints JSON to stdout.
func RunConfigDump() error {
	globalDir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := FindProjectRoot(cwd)
	m, err := config.LoadMerged(config.Paths{
		GlobalConfigDir: globalDir,
		ProjectRoot:     root,
	})
	if err != nil {
		return err
	}
	b, err := config.DumpJSON(m)
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(b))
	return err
}

// RunConfigSet writes a top-level key to the user config file (Phase 2 minimal; Phase 10 may extend).
func RunConfigSet(key, value string) error {
	globalDir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	return config.SetUserKey(globalDir, key, value)
}

// ApplyManagedEnvFromMerged sets process env from merged config "managed_env" (call only after trust).
// Returns keys set (for tests). Does not unset on missing key in map.
func ApplyManagedEnvFromMerged(m map[string]interface{}) []string {
	if m == nil {
		return nil
	}
	raw, ok := m["managed_env"]
	if !ok || raw == nil {
		return nil
	}
	mm, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	var keys []string
	for k, v := range mm {
		s, ok := v.(string)
		if !ok {
			continue
		}
		_ = os.Setenv(k, s)
		keys = append(keys, k)
	}
	return keys
}
