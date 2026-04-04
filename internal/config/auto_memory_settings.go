package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadTrustedAutoMemoryDirectory returns autoMemoryDirectory from trusted settings layers only,
// matching paths.ts getAutoMemPathSetting. Resolution order (first non-empty wins):
//
//  1. policySettings  → LoadPolicySettings(GlobalConfigDir): managed-settings.json (+ drop-ins),
//     RABBIT_CODE_POLICY_MDM_JSON, RABBIT_CODE_POLICY_REMOTE_JSON (see managed.go).
//  2. flagSettings    → p.FlagJSON, else env RABBIT_CODE_SETTINGS_JSON (same as merged config flag layer).
//  3. localSettings   → project-root LocalConfigFileName (.rabbit-code.local.json), gitignored;
//     Claude Code analogue: .claude/settings.local.json (not read here — rabbit paths only).
//  4. userSettings    → filepath.Join(GlobalConfigDir, UserConfigFileName) (config.json).
//
// projectSettings / .rabbit-code.json in the repo are intentionally NOT consulted — same security
// as TS excluding getSettingsForSource('projectSettings') for this key.
//
// The returned string is raw from JSON (trimmed). paths.validateMemoryPath(expandTilde: true) runs
// later in memdir.ResolveAutoMemDirWithOptions; invalid values are ignored and resolution falls
// through to the default projects/<sanitized-root>/memory/ layout, matching TS when
// getAutoMemPathSetting returns undefined.
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
