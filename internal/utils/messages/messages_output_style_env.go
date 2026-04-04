// Optional output style display-name parity: RABBIT_OUTPUT_STYLE_NAMES_JSON and
// RABBIT_OUTPUT_STYLE_CONFIG_PATH (JSON object like TS OUTPUT_STYLE_CONFIG).
package messages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var outputStyleMu sync.Mutex
var outputStyleCachedEnvRaw string
var outputStyleCachedEnvMap map[string]string

var outputStyleFileSig string
var outputStyleCachedFileMap map[string]string

func outputStyleNameFromEnv(style string) (string, bool) {
	raw := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_NAMES_JSON"))
	outputStyleMu.Lock()
	defer outputStyleMu.Unlock()
	if raw != outputStyleCachedEnvRaw {
		outputStyleCachedEnvRaw = raw
		outputStyleCachedEnvMap = nil
		if raw != "" {
			var m map[string]string
			if err := json.Unmarshal([]byte(raw), &m); err == nil {
				outputStyleCachedEnvMap = m
			}
		}
	}
	if len(outputStyleCachedEnvMap) == 0 {
		return "", false
	}
	n, ok := outputStyleCachedEnvMap[style]
	if !ok || strings.TrimSpace(n) == "" {
		return "", false
	}
	return n, true
}

func outputStyleNameFromConfigFile(style string) (string, bool) {
	path := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_CONFIG_PATH"))
	if path == "" {
		return "", false
	}
	path = filepath.Clean(path)
	st, err := os.Stat(path)
	if err != nil {
		outputStyleMu.Lock()
		outputStyleFileSig = ""
		outputStyleCachedFileMap = nil
		outputStyleMu.Unlock()
		return "", false
	}
	sig := fmt.Sprintf("%s|%d|%d", path, st.ModTime().UnixNano(), st.Size())

	outputStyleMu.Lock()
	defer outputStyleMu.Unlock()
	if sig != outputStyleFileSig {
		outputStyleFileSig = sig
		outputStyleCachedFileMap = nil
		b, err := os.ReadFile(path)
		if err != nil {
			return "", false
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(b, &root); err != nil {
			return "", false
		}
		m := make(map[string]string)
		for k, raw := range root {
			s := strings.TrimSpace(string(raw))
			if s == "" || s == "null" {
				continue
			}
			var name string
			if err := json.Unmarshal(raw, &name); err == nil && strings.TrimSpace(name) != "" {
				m[k] = strings.TrimSpace(name)
				continue
			}
			var obj struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(raw, &obj); err == nil && strings.TrimSpace(obj.Name) != "" {
				m[k] = strings.TrimSpace(obj.Name)
			}
		}
		outputStyleCachedFileMap = m
	}
	if len(outputStyleCachedFileMap) == 0 {
		return "", false
	}
	n, ok := outputStyleCachedFileMap[style]
	if !ok || strings.TrimSpace(n) == "" {
		return "", false
	}
	return n, true
}
