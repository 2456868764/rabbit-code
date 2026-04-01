package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeepMerge_nested(t *testing.T) {
	dst := map[string]interface{}{
		"templates": map[string]interface{}{"a": 1},
		"k":         "user",
	}
	src := map[string]interface{}{
		"templates": map[string]interface{}{"b": 2},
		"k":         "proj",
	}
	DeepMerge(dst, src)
	tm := dst["templates"].(map[string]interface{})
	if tm["a"].(int) != 1 || tm["b"].(int) != 2 {
		t.Fatalf("templates %+v", tm)
	}
	if dst["k"] != "proj" {
		t.Fatal(dst["k"])
	}
}

func TestLoadMerged_priorityUserProjectLocalFlag(t *testing.T) {
	root := t.TempDir()
	global := t.TempDir()
	user := filepath.Join(global, UserConfigFileName)
	if err := os.WriteFile(user, []byte(`{"k":"user","x":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(root, ".rabbit-code.json")
	if err := os.WriteFile(proj, []byte(`{"k":"project","y":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	loc := filepath.Join(root, LocalConfigFileName)
	if err := os.WriteFile(loc, []byte(`{"k":"local"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	p := Paths{
		GlobalConfigDir: global,
		ProjectRoot:     root,
		FlagJSON:        `{"k":"flag"}`,
	}
	m, err := LoadMerged(p)
	if err != nil {
		t.Fatal(err)
	}
	if m["k"] != "flag" {
		t.Fatalf("want flag got %v", m["k"])
	}
	if m["x"].(float64) != 1 || m["y"].(float64) != 2 {
		t.Fatalf("merged scalars %+v", m)
	}
}

func TestLoadMerged_localOverProject(t *testing.T) {
	root := t.TempDir()
	global := t.TempDir()
	user := filepath.Join(global, UserConfigFileName)
	_ = os.WriteFile(user, []byte(`{"a":1}`), 0o600)
	_ = os.WriteFile(filepath.Join(root, ".rabbit-code.json"), []byte(`{"b":2,"c":"proj"}`), 0o600)
	_ = os.WriteFile(filepath.Join(root, LocalConfigFileName), []byte(`{"c":"local"}`), 0o600)
	m, err := LoadMerged(Paths{GlobalConfigDir: global, ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if m["c"] != "local" {
		t.Fatalf("want local override got %v", m["c"])
	}
}

func TestLoadMerged_corruptUserJSON(t *testing.T) {
	global := t.TempDir()
	_ = os.WriteFile(filepath.Join(global, UserConfigFileName), []byte(`{not json`), 0o600)
	_, err := LoadMerged(Paths{GlobalConfigDir: global, ProjectRoot: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadMerged_policyOverFlag(t *testing.T) {
	root := t.TempDir()
	global := t.TempDir()
	_ = os.MkdirAll(filepath.Join(global, managedSettingsDropInDir), 0o755)
	_ = os.WriteFile(filepath.Join(global, managedSettingsFile), []byte(`{"k":"policy"}`), 0o600)
	m, err := LoadMerged(Paths{
		GlobalConfigDir: global,
		ProjectRoot:     root,
		FlagJSON:        `{"k":"flag"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if m["k"] != "policy" {
		t.Fatalf("policy should win over flag, got %v", m["k"])
	}
}

func TestAtomicWriteJSON_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, UserConfigFileName)
	m := map[string]interface{}{"hello": "world", "n": float64(3)}
	if err := AtomicWriteJSON(path, m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadJSONFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["hello"] != "world" || got["n"].(float64) != 3 {
		t.Fatalf("%+v", got)
	}
}

func TestValidate_managedEnv(t *testing.T) {
	errs := Validate(map[string]interface{}{
		"managed_env": map[string]interface{}{"OK": "v", "BAD": 1},
	})
	if len(errs) == 0 {
		t.Fatal("expected error")
	}
}

func TestSetUserKey(t *testing.T) {
	dir := t.TempDir()
	if err := SetUserKey(dir, "auto_theme", "dark"); err != nil {
		t.Fatal(err)
	}
	m, err := ReadJSONFile(filepath.Join(dir, UserConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if m["auto_theme"] != "dark" {
		t.Fatal(m)
	}
}

func TestMigrationExplicitVersion(t *testing.T) {
	root := t.TempDir()
	global := t.TempDir()
	_ = os.WriteFile(filepath.Join(global, UserConfigFileName), []byte(`{"schema_version":1}`), 0o600)
	m, err := LoadMerged(Paths{GlobalConfigDir: global, ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if m["schema_version"].(float64) != 2 {
		t.Fatalf("got %+v", m["schema_version"])
	}
	if m["migrated_from_v1"] != true {
		t.Fatal("migration marker missing")
	}
}

func TestDumpJSON(t *testing.T) {
	b, err := DumpJSON(map[string]interface{}{"a": "b"})
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMerged_pluginBaseLowestPriority(t *testing.T) {
	root := t.TempDir()
	global := t.TempDir()
	_ = os.WriteFile(filepath.Join(global, PluginBaseConfigFileName), []byte(`{"a":1,"k":"plugin"}`), 0o600)
	_ = os.WriteFile(filepath.Join(global, UserConfigFileName), []byte(`{"k":"user"}`), 0o600)
	m, err := LoadMerged(Paths{GlobalConfigDir: global, ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if m["k"] != "user" {
		t.Fatalf("user should override plugin base, got %v", m["k"])
	}
	if m["a"].(float64) != 1 {
		t.Fatal(m["a"])
	}
}

func TestLoadMerged_policyRemoteEnvWinsOverFile(t *testing.T) {
	t.Setenv(EnvPolicyRemoteJSON, `{"k":"remote"}`)
	root := t.TempDir()
	global := t.TempDir()
	_ = os.WriteFile(filepath.Join(global, UserConfigFileName), []byte(`{}`), 0o600)
	_ = os.WriteFile(filepath.Join(global, managedSettingsFile), []byte(`{"k":"file"}`), 0o600)
	m, err := LoadMerged(Paths{GlobalConfigDir: global, ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if m["k"] != "remote" {
		t.Fatalf("remote policy env should win, got %v", m["k"])
	}
}

func TestValidate_settingsSyncURLs(t *testing.T) {
	errs := Validate(map[string]interface{}{
		"download_user_settings_url": "not-a-url",
	})
	if len(errs) == 0 {
		t.Fatal("expected error")
	}
	errs = Validate(map[string]interface{}{
		"upload_user_settings_url": "https://example.com/sync",
	})
	if len(errs) != 0 {
		t.Fatal(errs)
	}
}
