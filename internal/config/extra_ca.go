package config

import (
	"path/filepath"
	"strings"
)

// ExtraCAPEMPaths extracts extra_ca_paths from merged settings and resolves relative paths
// against projectRoot, then cwd (when projectRoot is empty).
func ExtraCAPEMPaths(m map[string]interface{}, projectRoot, cwd string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m["extra_ca_paths"]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, el := range arr {
		s, ok := el.(string)
		if !ok {
			continue
		}
		p := resolveExtraCAPath(strings.TrimSpace(s), projectRoot, cwd)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func resolveExtraCAPath(p, projectRoot, cwd string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	base := projectRoot
	if strings.TrimSpace(base) == "" {
		base = cwd
	}
	if strings.TrimSpace(base) == "" {
		return filepath.Clean(p)
	}
	return filepath.Join(base, filepath.Clean(p))
}
