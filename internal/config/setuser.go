package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// SetUserKey sets a top-level key in the user config file (creates file if needed).
// Value is parsed as JSON when trimmed input starts with `{` or `[`; otherwise stored as a string.
func SetUserKey(globalConfigDir, key, value string) error {
	if globalConfigDir == "" || key == "" {
		return fmt.Errorf("config: globalConfigDir and key required")
	}
	path := filepath.Join(globalConfigDir, UserConfigFileName)
	m, err := ReadJSONFile(path)
	if err != nil {
		return err
	}
	val, err := parseSetValue(value)
	if err != nil {
		return err
	}
	m[key] = val
	if verrs := Validate(m); len(verrs) > 0 {
		return fmt.Errorf("validation: %v", verrs)
	}
	return AtomicWriteJSON(path, m)
}

func parseSetValue(s string) (interface{}, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		var v interface{}
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			return nil, err
		}
		return v, nil
	}
	return s, nil
}
