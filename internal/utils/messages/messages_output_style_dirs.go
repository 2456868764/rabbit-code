// Output style markdown dirs parity: TS getOutputStyleDirStyles (.claude/output-styles *.md).
package messages

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var outputStyleScanMu sync.Mutex
var outputStyleScanCachedFP string
var outputStyleScanCached map[string]string

func outputStyleScanDirsFingerprint() string {
	env := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_SCAN_DIRS"))
	if env == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(env)
	b.WriteByte('|')
	for _, d := range filepath.SplitList(env) {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		b.WriteString(outputStyleMarkdownDirFingerprint(d))
		b.WriteByte(';')
	}
	return b.String()
}

func outputStyleMarkdownDirFingerprint(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("err:%s", dir)
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%s|%d|", name, fi.ModTime().UnixNano())
	}
	return b.String()
}

func parseSimpleMarkdownFrontmatter(file []byte) (fields map[string]string, body []byte) {
	if !bytes.HasPrefix(file, []byte("---\n")) {
		return nil, file
	}
	rest := file[4:]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end < 0 {
		return nil, file
	}
	fmRaw := rest[:end]
	body = rest[end+len("\n---\n"):]
	fields = make(map[string]string)
	for _, line := range bytes.Split(fmRaw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(string(parts[0]))
		v := strings.TrimSpace(string(parts[1]))
		v = strings.Trim(v, `"'`)
		fields[k] = v
	}
	return fields, body
}

func mergeMarkdownOutputStyleNamesFromDir(dst map[string]string, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		fm, _ := parseSimpleMarkdownFrontmatter(b)
		key := stem
		display := stem
		if fm != nil {
			if n := strings.TrimSpace(fm["name"]); n != "" {
				key = n
				display = n
			}
		}
		if key != "" {
			dst[key] = display
		}
	}
}

func outputStyleNamesFromScanDirs() map[string]string {
	out := make(map[string]string)
	env := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_SCAN_DIRS"))
	if env == "" {
		return out
	}
	for _, d := range filepath.SplitList(env) {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		mergeMarkdownOutputStyleNamesFromDir(out, d)
	}
	return out
}

func outputStyleNameFromScanDirs(style string) (string, bool) {
	outputStyleScanMu.Lock()
	defer outputStyleScanMu.Unlock()
	fp := outputStyleScanDirsFingerprint()
	if fp == "" {
		outputStyleScanCachedFP = ""
		outputStyleScanCached = nil
		return "", false
	}
	if fp != outputStyleScanCachedFP {
		outputStyleScanCachedFP = fp
		outputStyleScanCached = outputStyleNamesFromScanDirs()
	}
	if len(outputStyleScanCached) == 0 {
		return "", false
	}
	n, ok := outputStyleScanCached[style]
	if !ok || strings.TrimSpace(n) == "" {
		return "", false
	}
	return n, true
}
