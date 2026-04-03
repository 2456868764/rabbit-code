package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadTrustedAutoMemoryDirectory returns autoMemoryDirectory from trusted settings layers only,
// matching paths.ts getAutoMemPathSetting (policy → flag → local → user). Project settings
// (.rabbit-code.json) are never read so a repo cannot redirect auto-memory via this key.
func LoadTrustedAutoMemoryDirectory(p Paths) (string, error) {
	pol, err := LoadPolicySettings(p.GlobalConfigDir)
	if err != nil {
		return "", err
	}
	if s := autoMemoryDirectoryFromMap(pol); s != "" {
		return s, nil
	}

	flagSrc := p.FlagJSON
	if flagSrc == "" {
		flagSrc = os.Getenv(EnvFlagJSON)
	}
	if strings.TrimSpace(flagSrc) != "" {
		var flag map[string]interface{}
		if err := json.Unmarshal([]byte(flagSrc), &flag); err != nil {
			return "", fmt.Errorf("flag settings (%s): %w", EnvFlagJSON, err)
		}
		if s := autoMemoryDirectoryFromMap(flag); s != "" {
			return s, nil
		}
	}

	loc, err := ReadJSONFile(p.resolvedLocal())
	if err != nil {
		return "", err
	}
	if s := autoMemoryDirectoryFromMap(loc); s != "" {
		return s, nil
	}

	user, err := ReadJSONFile(p.resolvedUser())
	if err != nil {
		return "", err
	}
	if s := autoMemoryDirectoryFromMap(user); s != "" {
		return s, nil
	}

	return "", nil
}

func autoMemoryDirectoryFromMap(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	v, ok := m["autoMemoryDirectory"]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
