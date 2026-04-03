package engine

import (
	"github.com/2456868764/rabbit-code/internal/config"
)

// ApplyMergedSettingsForMemdir sets InitialSettings (shallow copy) and MemdirTrustedAutoMemoryDirectory on cfg
// from merged settings and trusted layers (paths.ts H8). merged may be nil; trusted path is still loaded from disk.
func ApplyMergedSettingsForMemdir(cfg *Config, p config.Paths, merged map[string]interface{}) error {
	if cfg == nil {
		return nil
	}
	if merged != nil {
		cfg.InitialSettings = shallowCloneSettingsMap(merged)
	}
	trusted, err := config.LoadTrustedAutoMemoryDirectory(p)
	if err != nil {
		return err
	}
	cfg.MemdirTrustedAutoMemoryDirectory = trusted
	return nil
}

func shallowCloneSettingsMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
