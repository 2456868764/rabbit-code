package app

import (
	"log/slog"
	"os"
)

// EnvUndercover mirrors utils/undercover.ts / setup.ts (non-feature() gate).
const EnvUndercover = "CLAUDE_CODE_UNDERCOVER"

// LogUndercoverMode emits a single debug line when CLAUDE_CODE_UNDERCOVER is set.
// Full OSS attribution behavior is out of scope for Phase 1; see PHASE01_SPEC §4.
func LogUndercoverMode(log *slog.Logger) {
	if log == nil || !truthy(os.Getenv(EnvUndercover)) {
		return
	}
	log.Debug("undercover mode env set (PARITY: restored-src utils/undercover.ts; Phase 1 logs only)")
}
