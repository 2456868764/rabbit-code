package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// loadManagedPolicyFiles merges managed-settings.json then managed-settings.d/*.json (sorted by name).
func loadManagedPolicyFiles(configDir string) (map[string]interface{}, error) {
	if configDir == "" {
		return map[string]interface{}{}, nil
	}
	out := map[string]interface{}{}
	basePath := filepath.Join(configDir, managedSettingsFile)
	base, err := ReadJSONFile(basePath)
	if err != nil {
		return nil, err
	}
	DeepMerge(out, base)

	dropDir := filepath.Join(configDir, managedSettingsDropInDir)
	entries, err := os.ReadDir(dropDir)
	if err != nil {
		if errorsIsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := filepath.Join(dropDir, name)
		frag, err := ReadJSONFile(p)
		if err != nil {
			return nil, err
		}
		DeepMerge(out, frag)
	}
	return out, nil
}

// LoadPolicySettings builds the policy stack: HKCU (stub) → managed files → MDM env → remote env (P2.3.1).
// Later layers override earlier layers (DeepMerge).
func LoadPolicySettings(configDir string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	DeepMerge(out, policyHKCUSnapshot())

	files, err := loadManagedPolicyFiles(configDir)
	if err != nil {
		return nil, err
	}
	DeepMerge(out, files)

	if s := strings.TrimSpace(os.Getenv(EnvPolicyMDMJSON)); s != "" {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return nil, &policyEnvError{name: EnvPolicyMDMJSON, err: err}
		}
		if m != nil {
			DeepMerge(out, m)
		}
	}
	if s := strings.TrimSpace(os.Getenv(EnvPolicyRemoteJSON)); s != "" {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return nil, &policyEnvError{name: EnvPolicyRemoteJSON, err: err}
		}
		if m != nil {
			DeepMerge(out, m)
		}
	}
	return out, nil
}

type policyEnvError struct {
	name string
	err  error
}

func (e *policyEnvError) Error() string {
	return "policy " + e.name + ": " + e.err.Error()
}

func (e *policyEnvError) Unwrap() error {
	return e.err
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
