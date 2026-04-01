package app

import (
	"log/slog"

	"github.com/2456868764/rabbit-code/internal/features"
)

// runLodestonePhase1Hook documents the interactive custom-protocol registration order.
// Phase 12 wires real bridge handlers; Phase 1 only logs when LODESTONE is enabled.
func runLodestonePhase1Hook(log *slog.Logger, nonInteractive bool) {
	if log == nil || !features.LodestoneEnabled() {
		return
	}
	if nonInteractive {
		log.Debug("LODESTONE: protocol registration skipped (non-interactive)")
		return
	}
	log.Debug("LODESTONE: custom protocol registration placeholder (Phase 12: internal/bridge)")
}
