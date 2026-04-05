package webfetchtool

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// persistBinaryWebFetch mirrors persistBinaryContent for WebFetch (tool-results dir).
func persistBinaryWebFetch(dir string, body []byte, contentType string) (absPath string, size int, err error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}
	id := fmt.Sprintf("webfetch-%d-%s", time.Now().UnixNano(), randomHex(4))
	name := fmt.Sprintf("%s.%s", id, extensionForMimeType(contentType))
	absPath = filepath.Join(dir, name)
	if err := os.WriteFile(absPath, body, 0o644); err != nil {
		return "", 0, err
	}
	return absPath, len(body), nil
}
