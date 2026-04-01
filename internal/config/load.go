package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/migrations"
)

// Paths holds resolved config file paths (from app/discovery).
type Paths struct {
	GlobalConfigDir   string
	UserFile          string // filepath.Join(GlobalConfigDir, UserConfigFileName) if empty, computed
	ProjectRoot       string
	ProjectFile       string // .rabbit-code.json
	LocalFile         string // .rabbit-code.local.json
	FlagJSON          string // raw JSON from CLI or env; if empty, os.Getenv(EnvFlagJSON)
}

func (p Paths) resolvedUser() string {
	if p.UserFile != "" {
		return p.UserFile
	}
	if p.GlobalConfigDir == "" {
		return ""
	}
	return filepath.Join(p.GlobalConfigDir, UserConfigFileName)
}

func (p Paths) resolvedProject() string {
	if p.ProjectFile != "" {
		return p.ProjectFile
	}
	if p.ProjectRoot == "" {
		return ""
	}
	return filepath.Join(p.ProjectRoot, ".rabbit-code.json")
}

func (p Paths) resolvedLocal() string {
	if p.LocalFile != "" {
		return p.LocalFile
	}
	if p.ProjectRoot == "" {
		return ""
	}
	return filepath.Join(p.ProjectRoot, LocalConfigFileName)
}

// LoadMerged loads and merges: plugin base < user < project < local < flag < policy stack; then migrations.Apply; then Validate.
func LoadMerged(p Paths) (map[string]interface{}, error) {
	pluginPath := ""
	if p.GlobalConfigDir != "" {
		pluginPath = filepath.Join(p.GlobalConfigDir, PluginBaseConfigFileName)
	}
	plugin, err := ReadJSONFile(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("plugin base settings: %w", err)
	}
	user, err := ReadJSONFile(p.resolvedUser())
	if err != nil {
		return nil, fmt.Errorf("user settings: %w", err)
	}
	proj, err := ReadJSONFile(p.resolvedProject())
	if err != nil {
		return nil, fmt.Errorf("project settings: %w", err)
	}
	loc, err := ReadJSONFile(p.resolvedLocal())
	if err != nil {
		return nil, fmt.Errorf("local settings: %w", err)
	}

	out := map[string]interface{}{}
	DeepMerge(out, plugin)
	DeepMerge(out, user)
	DeepMerge(out, proj)
	DeepMerge(out, loc)

	flagSrc := p.FlagJSON
	if flagSrc == "" {
		flagSrc = os.Getenv(EnvFlagJSON)
	}
	if strings.TrimSpace(flagSrc) != "" {
		var flag map[string]interface{}
		if err := json.Unmarshal([]byte(flagSrc), &flag); err != nil {
			return nil, fmt.Errorf("flag settings (%s): %w", EnvFlagJSON, err)
		}
		if flag != nil {
			DeepMerge(out, flag)
		}
	}

	pol, err := LoadPolicySettings(p.GlobalConfigDir)
	if err != nil {
		return nil, fmt.Errorf("policy settings: %w", err)
	}
	DeepMerge(out, pol)

	if err := migrations.Apply(out); err != nil {
		return nil, err
	}
	if verrs := Validate(out); len(verrs) > 0 {
		return nil, fmt.Errorf("validation: %v", verrs)
	}
	return out, nil
}
