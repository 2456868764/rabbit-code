// Plugin output-styles parity: TS loadPluginOutputStyles / walkPluginMarkdown (recursive *.md).
package messages

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type outputStylePluginSpec struct {
	Plugin string `json:"plugin"`
	Dir    string `json:"dir"`
}

var outputStylePluginMu sync.Mutex
var outputStylePluginCachedSig string
var outputStylePluginCached map[string]string

func outputStylePluginLoadSpecs() ([]outputStylePluginSpec, error) {
	var specs []outputStylePluginSpec
	if p := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_PLUGINS_PATH")); p != "" {
		b, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			return nil, err
		}
		var fromFile []outputStylePluginSpec
		if err := json.Unmarshal(b, &fromFile); err != nil {
			return nil, err
		}
		specs = append(specs, fromFile...)
	}
	if raw := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_PLUGINS_JSON")); raw != "" {
		var fromEnv []outputStylePluginSpec
		if err := json.Unmarshal([]byte(raw), &fromEnv); err != nil {
			return nil, err
		}
		specs = append(specs, fromEnv...)
	}
	return specs, nil
}

func outputStylePluginSpecsFingerprint(specs []outputStylePluginSpec) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_PLUGINS_JSON")))
	b.WriteByte('|')
	if p := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_PLUGINS_PATH")); p != "" {
		p = filepath.Clean(p)
		if st, err := os.Stat(p); err == nil {
			fmt.Fprintf(&b, "%s|%d|%d", p, st.ModTime().UnixNano(), st.Size())
		} else {
			b.WriteString("!")
		}
	}
	for _, sp := range specs {
		d := strings.TrimSpace(sp.Dir)
		b.WriteString(";")
		b.WriteString(sp.Plugin)
		b.WriteByte('=')
		b.WriteString(d)
		_ = filepath.WalkDir(d, func(p string, ent fs.DirEntry, err error) error {
			if err != nil || ent.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(ent.Name()), ".md") {
				return nil
			}
			fi, err := ent.Info()
			if err != nil {
				return nil
			}
			fmt.Fprintf(&b, "|%s|%d", p, fi.ModTime().UnixNano())
			return nil
		})
	}
	return b.String()
}

func mergePluginOutputStyleMarkdown(dst map[string]string, pluginName, root string) {
	pn := strings.TrimSpace(pluginName)
	if pn == "" {
		pn = "plugin"
	}
	_ = filepath.WalkDir(root, func(path string, ent fs.DirEntry, err error) error {
		if err != nil || ent.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(ent.Name()), ".md") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		stem := strings.TrimSuffix(ent.Name(), filepath.Ext(ent.Name()))
		fm, _ := parseSimpleMarkdownFrontmatter(b)
		base := stem
		if fm != nil {
			if n := strings.TrimSpace(fm["name"]); n != "" {
				base = n
			}
		}
		key := pn + ":" + base
		if key != "" {
			// TS OutputStyleConfig.name is namespaced; messages.ts uses that string in the reminder.
			dst[key] = key
		}
		return nil
	})
}

func outputStyleNamesFromPluginSpecs(specs []outputStylePluginSpec) map[string]string {
	out := make(map[string]string)
	for _, sp := range specs {
		d := strings.TrimSpace(sp.Dir)
		if d == "" {
			continue
		}
		mergePluginOutputStyleMarkdown(out, sp.Plugin, d)
	}
	return out
}

func outputStyleNameFromPlugins(style string) (string, bool) {
	specs, err := outputStylePluginLoadSpecs()
	if err != nil || len(specs) == 0 {
		outputStylePluginMu.Lock()
		outputStylePluginCachedSig = ""
		outputStylePluginCached = nil
		outputStylePluginMu.Unlock()
		return "", false
	}
	fp := outputStylePluginSpecsFingerprint(specs)
	outputStylePluginMu.Lock()
	defer outputStylePluginMu.Unlock()
	if fp != outputStylePluginCachedSig {
		outputStylePluginCachedSig = fp
		outputStylePluginCached = outputStyleNamesFromPluginSpecs(specs)
	}
	if len(outputStylePluginCached) == 0 {
		return "", false
	}
	n, ok := outputStylePluginCached[style]
	if !ok || strings.TrimSpace(n) == "" {
		return "", false
	}
	return n, true
}
