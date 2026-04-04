// Optional OUTPUT_STYLE_CONFIG parity via RABBIT_OUTPUT_STYLE_NAMES_JSON ({"StyleKey":"Display Name"}).
package messages

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

var outputStyleMu sync.Mutex
var outputStyleCachedRaw string
var outputStyleCachedMap map[string]string

func outputStyleNameFromEnv(style string) (string, bool) {
	raw := strings.TrimSpace(os.Getenv("RABBIT_OUTPUT_STYLE_NAMES_JSON"))
	outputStyleMu.Lock()
	defer outputStyleMu.Unlock()
	if raw != outputStyleCachedRaw {
		outputStyleCachedRaw = raw
		outputStyleCachedMap = nil
		if raw != "" {
			var m map[string]string
			if err := json.Unmarshal([]byte(raw), &m); err == nil {
				outputStyleCachedMap = m
			}
		}
	}
	if len(outputStyleCachedMap) == 0 {
		return "", false
	}
	n, ok := outputStyleCachedMap[style]
	if !ok || strings.TrimSpace(n) == "" {
		return "", false
	}
	return n, true
}
