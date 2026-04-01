package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// ReadJSONFile parses a single JSON object from path. Missing file returns empty map, nil.
func ReadJSONFile(path string) (map[string]interface{}, error) {
	if path == "" {
		return map[string]interface{}{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	if len(b) == 0 {
		return map[string]interface{}{}, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if m == nil {
		m = map[string]interface{}{}
	}
	return m, nil
}
