package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteJSON writes pretty-printed JSON to path using rename-over (AC2-1 / P2.1.2).
func AtomicWriteJSON(path string, m map[string]interface{}) error {
	if path == "" {
		return fmt.Errorf("config: empty path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(dir, ".rabbit-code-config-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// DumpJSON returns indented JSON for merged map (E2E dump).
func DumpJSON(m map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
