package bashtool

import (
	"os"
	"strconv"
	"strings"
)

// Default and max bash timeouts mirror utils/timeouts.ts (BASH_DEFAULT_TIMEOUT_MS / BASH_MAX_TIMEOUT_MS).
const (
	defaultBashTimeoutMS = 120_000
	maxBashTimeoutMSCap  = 600_000
)

// DefaultBashTimeoutMs mirrors getDefaultBashTimeoutMs.
func DefaultBashTimeoutMs() int {
	if v := strings.TrimSpace(os.Getenv("BASH_DEFAULT_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultBashTimeoutMS
}

// MaxBashTimeoutMs mirrors getMaxBashTimeoutMs.
func MaxBashTimeoutMs() int {
	def := DefaultBashTimeoutMs()
	if v := strings.TrimSpace(os.Getenv("BASH_MAX_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n < def {
				return def
			}
			return n
		}
	}
	if maxBashTimeoutMSCap < def {
		return def
	}
	return maxBashTimeoutMSCap
}
