package fileedittool

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed settings_schema.json
var settingsSchemaJSON []byte

// settingsSchemaID matches settings_schema.json $id (SchemaStore claude-code-settings).
const settingsSchemaID = "https://json.schemastore.org/claude-code-settings.json"

var settingsValidator struct {
	once    sync.Once
	schema  *jsonschema.Schema
	topKeys map[string]struct{}
	err     error
}

func ensureSettingsValidator() (*jsonschema.Schema, map[string]struct{}, error) {
	settingsValidator.once.Do(func() {
		var doc map[string]any
		if err := json.Unmarshal(settingsSchemaJSON, &doc); err != nil {
			settingsValidator.err = fmt.Errorf("fileedittool: parse embedded settings schema: %w", err)
			return
		}
		props, _ := doc["properties"].(map[string]any)
		if len(props) == 0 {
			settingsValidator.err = errors.New("fileedittool: settings schema has no properties")
			return
		}
		settingsValidator.topKeys = make(map[string]struct{}, len(props))
		for k := range props {
			settingsValidator.topKeys[k] = struct{}{}
		}
		c := jsonschema.NewCompiler()
		c.UseRegexpEngine(settingsSchemaRegexpCompile)
		if err := c.AddResource(settingsSchemaID, doc); err != nil {
			settingsValidator.err = fmt.Errorf("fileedittool: add settings schema resource: %w", err)
			return
		}
		settingsValidator.schema, settingsValidator.err = c.Compile(settingsSchemaID)
	})
	if settingsValidator.err != nil {
		return nil, nil, settingsValidator.err
	}
	return settingsValidator.schema, settingsValidator.topKeys, nil
}

// isClaudeSettingsPath is a simplified mirror of permissions/filesystem.ts isClaudeSettingsPath
// (endsWith .claude/settings.json|.local.json, case-insensitive, cleaned path).
func isClaudeSettingsPath(abs string) bool {
	p := filepath.Clean(abs)
	pl := strings.ToLower(filepath.ToSlash(p))
	return strings.HasSuffix(pl, "/.claude/settings.json") ||
		strings.HasSuffix(pl, "/.claude/settings.local.json")
}

func validateSettingsContentJSON(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("empty document")
	}
	sch, topKeys, err := ensureSettingsValidator()
	if err != nil {
		return err
	}
	var instance map[string]any
	if err := json.Unmarshal([]byte(s), &instance); err != nil {
		return fmt.Errorf("Invalid JSON: %w", err)
	}
	if err := sch.Validate(instance); err != nil {
		return fmt.Errorf("Settings validation failed:\n%s", err.Error())
	}
	for k := range instance {
		if _, ok := topKeys[k]; !ok {
			return fmt.Errorf("Settings validation failed:\n- : Unrecognized field: %s", k)
		}
	}
	return nil
}

// validateSettingsFileEdit mirrors validateInputForSettingsFileEdit: if the file was schema-valid
// before, the post-edit content must stay valid. Schema: SchemaStore claude-code-settings plus
// top-level key allowlist (Zod .strict() parity; SchemaStore root allows additionalProperties).
func validateSettingsFileEdit(abs, originalContent string, updatedContent string) error {
	if !isClaudeSettingsPath(abs) {
		return nil
	}
	orig := strings.TrimSpace(originalContent)
	if orig == "" {
		return nil
	}
	if !json.Valid([]byte(orig)) {
		return nil
	}
	if err := validateSettingsContentJSON(orig); err != nil {
		return nil
	}
	if err := validateSettingsContentJSON(updatedContent); err != nil {
		return fmt.Errorf("Claude Code settings.json validation failed after edit:\n%s\n\nFull schema:\n%s\n\nIMPORTANT: Do not update the env unless explicitly instructed to do so.", err.Error(), string(settingsSchemaJSON))
	}
	return nil
}
