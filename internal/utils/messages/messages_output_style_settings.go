// Settings outputStyle fallback parity: TS getOutputStyleConfig uses settings.outputStyle when the
// client omits a concrete attachment style (we only need the key for the reminder line).
package messages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// settingsFallbackOutputStyle returns RABBIT_SETTINGS_OUTPUT_STYLE, else outputStyle from
// RABBIT_CLAUDE_SETTINGS_PATH JSON (Claude Code settings file shape).
func settingsFallbackOutputStyle() string {
	if s := strings.TrimSpace(os.Getenv("RABBIT_SETTINGS_OUTPUT_STYLE")); s != "" {
		return s
	}
	path := strings.TrimSpace(os.Getenv("RABBIT_CLAUDE_SETTINGS_PATH"))
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	v, ok := m["outputStyle"]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case fmt.Stringer:
		return strings.TrimSpace(x.String())
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}
