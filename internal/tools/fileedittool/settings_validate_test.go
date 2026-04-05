package fileedittool

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSettingsFileEdit_notSettingsPath(t *testing.T) {
	if err := validateSettingsFileEdit("/tmp/other.json", `{}`, `{`); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSettingsFileEdit_invalidBeforeAllowsEdit(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, ".claude", "settings.json")
	if err := validateSettingsFileEdit(abs, `{`, `{`); err != nil {
		t.Fatal(err)
	}
	if err := validateSettingsFileEdit(abs, `{not json`, `{"autoUpdatesChannel":"stable"}`); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSettingsFileEdit_validBeforeRejectsBadAfter(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, ".claude", "settings.json")
	before := `{"autoUpdatesChannel": "stable"}`
	after := `{"autoUpdatesChannel": "stable", "unknownTopLevelKey": true}`
	err := validateSettingsFileEdit(abs, before, after)
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "Unrecognized field: unknownTopLevelKey") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateSettingsFileEdit_schemaNestedError(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, ".claude", "settings.json")
	before := `{"autoUpdatesChannel": "stable"}`
	after := `{"autoUpdatesChannel": "stable", "env": {"lowercase": "x"}}`
	err := validateSettingsFileEdit(abs, before, after)
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateSettingsFileEdit_happyPath(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, ".claude", "settings.json")
	s := `{"autoUpdatesChannel": "stable", "outputStyle": "default"}`
	if err := validateSettingsFileEdit(abs, s, s); err != nil {
		t.Fatal(err)
	}
}
