package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadTemplateMarkdownAppendix reads <dir>/<name>.md for each name and returns a single appendix string (P5.F.7 body load).
func LoadTemplateMarkdownAppendix(dir string, names []string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", nil
	}
	var b strings.Builder
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		if strings.Contains(name, string(os.PathSeparator)) || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			return "", fmt.Errorf("query: invalid template name %q", rawName)
		}
		p := filepath.Join(dir, name+".md")
		raw, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "\n\n## Template %s\n%s", name, string(raw))
	}
	return b.String(), nil
}
