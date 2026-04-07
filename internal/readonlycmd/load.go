package readonlycmd

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed allowlist_shared.json
var allowlistSharedJSON []byte

//go:embed allowlist_extras.json
var allowlistExtrasJSON []byte

var (
	loadOnce sync.Once
	merged   map[string]CommandConfig
)

func mergedAllowlist() map[string]CommandConfig {
	loadOnce.Do(func() {
		merged = map[string]CommandConfig{}
		loadJSONInto(merged, allowlistSharedJSON)
		loadJSONInto(merged, allowlistExtrasJSON)
	})
	return merged
}

type fileEntry struct {
	SafeFlags          map[string]string `json:"safeFlags"`
	RespectsDoubleDash *bool             `json:"respectsDoubleDash"`
}

func loadJSONInto(dst map[string]CommandConfig, raw []byte) {
	var f map[string]fileEntry
	if err := json.Unmarshal(raw, &f); err != nil {
		panic("readonlycmd: invalid embedded allowlist: " + err.Error())
	}
	for k, v := range f {
		cfg := CommandConfig{SafeFlags: map[string]ArgType{}, RespectsDoubleDash: true}
		if v.RespectsDoubleDash != nil {
			cfg.RespectsDoubleDash = *v.RespectsDoubleDash
		}
		for fk, ft := range v.SafeFlags {
			cfg.SafeFlags[fk] = ArgType(ft)
		}
		dst[k] = cfg
	}
}
