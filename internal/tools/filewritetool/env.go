package filewritetool

import (
	"os"
	"strings"
)

// envTruthy mirrors isEnvTruthy(process.env.CLAUDE_CODE_REMOTE) for optional gitDiff attachment.
func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
