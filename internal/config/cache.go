package config

import (
	"fmt"
	"os"
	pathpkg "path/filepath"
	"strings"
)

// Loader reloads when any contributing file mtime or flag env changes.
type Loader struct {
	Paths Paths
	last  string
	data  map[string]interface{}
}

// Load returns cached merged settings if inputs unchanged.
func (l *Loader) Load() (map[string]interface{}, error) {
	fp, err := fingerprint(l.Paths)
	if err != nil {
		return nil, err
	}
	if fp == l.last && l.data != nil {
		return l.data, nil
	}
	m, err := LoadMerged(l.Paths)
	if err != nil {
		return nil, err
	}
	l.last = fp
	l.data = m
	return m, nil
}

// Invalidate clears the cache (e.g. after external write).
func (l *Loader) Invalidate() {
	l.last = ""
	l.data = nil
}

func fingerprint(p Paths) (string, error) {
	var b strings.Builder
	appendFile := func(path string) {
		if path == "" {
			return
		}
		st, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				b.WriteString(path)
				b.WriteString(":missing\n")
				return
			}
			return
		}
		fmt.Fprintf(&b, "%s:%d:%d\n", path, st.Size(), st.ModTime().UnixNano())
	}

	appendFile(p.resolvedUser())
	appendFile(p.resolvedProject())
	appendFile(p.resolvedLocal())
	if p.GlobalConfigDir != "" {
		appendFile(pathpkg.Join(p.GlobalConfigDir, PluginBaseConfigFileName))
	}
	appendFile(pathpkg.Join(p.GlobalConfigDir, managedSettingsFile))

	dropDir := pathpkg.Join(p.GlobalConfigDir, managedSettingsDropInDir)
	entries, _ := os.ReadDir(dropDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".") {
			continue
		}
		appendFile(pathpkg.Join(dropDir, name))
	}

	flag := p.FlagJSON
	if flag == "" {
		flag = os.Getenv(EnvFlagJSON)
	}
	fmt.Fprintf(&b, "flag:%q\n", flag)
	fmt.Fprintf(&b, "policyMDM:%q\n", os.Getenv(EnvPolicyMDMJSON))
	fmt.Fprintf(&b, "policyRemote:%q\n", os.Getenv(EnvPolicyRemoteJSON))
	return b.String(), nil
}
