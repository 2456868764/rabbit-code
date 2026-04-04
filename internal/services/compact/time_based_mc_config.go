package compact

import "github.com/2456868764/rabbit-code/internal/features"

// TimeBasedMCConfig mirrors timeBasedMCConfig.ts TimeBasedMCConfig (GrowthBook tengu_slate_heron).
type TimeBasedMCConfig struct {
	// Enabled master switch; default false in TS.
	Enabled bool
	// GapThresholdMinutes triggers when (now − last assistant) exceeds this; default 60.
	GapThresholdMinutes int
	// KeepRecent compactable tool results to retain when clearing; default 5.
	KeepRecent int
}

// GetTimeBasedMCConfig mirrors getTimeBasedMCConfig(): always returns defaults merged with env
// (headless analogue of getFeatureValue_CACHED_MAY_BE_STALE('tengu_slate_heron', defaults)).
func GetTimeBasedMCConfig() TimeBasedMCConfig {
	return TimeBasedMCConfig{
		Enabled:             features.TimeBasedMicrocompactEnabled(),
		GapThresholdMinutes: features.TimeBasedMCGapThresholdMinutes(),
		KeepRecent:          features.TimeBasedMCKeepRecent(),
	}
}
