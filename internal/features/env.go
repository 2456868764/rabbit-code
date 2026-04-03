// Package features holds runtime feature flags aligned with SOURCE_FEATURE_FLAGS.md.
// Core env toggles: env.go (Phase-1-style). API client + headless query flags: rabbit_env.go.
// Later phases may merge with config/telemetry.
package features

import (
	"os"
	"strconv"
	"strings"
)

// Environment variable names (rabbit-code convention).
const (
	EnvHardFail             = "RABBIT_CODE_HARD_FAIL"
	EnvSlowOperationLogging = "RABBIT_CODE_SLOW_OPERATION_LOGGING"
	EnvFilePersistence      = "RABBIT_CODE_FILE_PERSISTENCE"
	EnvLodestone            = "RABBIT_CODE_LODESTONE"
)

func truthy(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1", "true", "yes", "on":
		return true
	}
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return false
}

// HardFailEnabled mirrors feature('HARD_FAIL'): treat bootstrap failures as fatal-style (non-zero exit, stderr prefix).
func HardFailEnabled() bool {
	return truthy(os.Getenv(EnvHardFail))
}

// SlowOperationLoggingEnabled mirrors feature('SLOW_OPERATION_LOGGING'): separate slow-op log sink when true.
func SlowOperationLoggingEnabled() bool {
	return truthy(os.Getenv(EnvSlowOperationLogging))
}

// FilePersistenceEnabled mirrors feature('FILE_PERSISTENCE'): register startup/shutdown hooks (Phase 1 no-op body).
func FilePersistenceEnabled() bool {
	return truthy(os.Getenv(EnvFilePersistence))
}

// LodestoneEnabled mirrors feature('LODESTONE'): interactive-only bootstrap hook until Phase 12 registers protocols.
func LodestoneEnabled() bool {
	return truthy(os.Getenv(EnvLodestone))
}
