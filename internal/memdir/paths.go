package memdir

import (
	"bytes"
	"os"
)

// SessionFragmentsFromPaths reads each path as UTF-8 text and returns non-empty trimmed fragments.
// Empty files are skipped. totalRawBytes is the sum of on-disk file sizes (before trim) for attachment-style budgets.
func SessionFragmentsFromPaths(paths []string) (fragments []string, totalRawBytes int, err error) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, 0, err
		}
		totalRawBytes += len(b)
		s := string(bytes.TrimSpace(b))
		if s != "" {
			fragments = append(fragments, s)
		}
	}
	return fragments, totalRawBytes, nil
}
