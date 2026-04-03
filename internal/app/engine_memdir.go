package app

import (
	"github.com/2456868764/rabbit-code/internal/config"
	"github.com/2456868764/rabbit-code/internal/query/engine"
)

// ApplyEngineMemdirFromRuntime fills cfg.InitialSettings and cfg.MemdirTrustedAutoMemoryDirectory from rt
// (after LoadAndApplyMergedConfig). Safe when rt or cfg is nil.
func ApplyEngineMemdirFromRuntime(rt *Runtime, cfg *engine.Config) error {
	if rt == nil || cfg == nil {
		return nil
	}
	root := ""
	if rt.State != nil {
		root = rt.State.ProjectRoot()
	}
	return engine.ApplyMergedSettingsForMemdir(cfg, config.Paths{
		GlobalConfigDir: rt.GlobalConfigDir,
		ProjectRoot:     root,
	}, rt.MergedSettings)
}
