package memdir

import (
	"bytes"
	"os"
)

// SessionFragmentsFromPaths reads each path as UTF-8 text and returns non-empty trimmed lines as fragments.
// Empty files are skipped. Used for memdir-style session injection (Phase 5 stub).
func SessionFragmentsFromPaths(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		if p == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		s := string(bytes.TrimSpace(b))
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}
