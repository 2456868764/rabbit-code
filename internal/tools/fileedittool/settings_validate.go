package fileedittool

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// isClaudeSettingsPath is a simplified mirror of permissions/filesystem.ts isClaudeSettingsPath
// (endsWith .claude/settings.json|.local.json, case-insensitive, cleaned path).
func isClaudeSettingsPath(abs string) bool {
	p := filepath.Clean(abs)
	pl := strings.ToLower(filepath.ToSlash(p))
	return strings.HasSuffix(pl, "/.claude/settings.json") ||
		strings.HasSuffix(pl, "/.claude/settings.local.json")
}

// validateSettingsFileEdit mirrors validateInputForSettingsFileEdit using JSON syntax check (full SettingsSchema is TS/AJV).
func validateSettingsFileEdit(abs, originalContent string, updatedContent string) error {
	if !isClaudeSettingsPath(abs) {
		return nil
	}
	orig := strings.TrimSpace(originalContent)
	if orig == "" || !json.Valid([]byte(orig)) {
		return nil
	}
	if json.Valid([]byte(updatedContent)) {
		return nil
	}
	return fmt.Errorf("Claude Code settings.json validation failed after edit: result is not valid JSON.\n\nIMPORTANT: Do not update the env unless explicitly instructed to do so.")
}
