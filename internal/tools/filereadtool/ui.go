package filereadtool

import (
	"path/filepath"
	"strings"
)

// UserFacingName mirrors UI.tsx userFacingName (plans directory / agent output branches deferred).
func UserFacingName(filePath string) string {
	if filePath == "" {
		return "Read"
	}
	return "Read"
}

// GetToolUseSummary mirrors UI.tsx getToolUseSummary (agent task id branch deferred).
func GetToolUseSummary(filePath string) string {
	if strings.TrimSpace(filePath) == "" {
		return ""
	}
	return filepath.Base(filePath)
}
