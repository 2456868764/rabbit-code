package compact

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestGetTimeBasedMCConfig_defaults(t *testing.T) {
	t.Setenv(features.EnvTimeBasedMicrocompact, "")
	t.Setenv(features.EnvTimeBasedMCGapMinutes, "")
	t.Setenv(features.EnvTimeBasedMCKeepRecent, "")
	c := GetTimeBasedMCConfig()
	if c.Enabled {
		t.Fatal("enabled default off")
	}
	if c.GapThresholdMinutes != 60 || c.KeepRecent != 5 {
		t.Fatalf("%+v", c)
	}
}

func TestGetTimeBasedMCConfig_env(t *testing.T) {
	t.Setenv(features.EnvTimeBasedMicrocompact, "1")
	t.Setenv(features.EnvTimeBasedMCGapMinutes, "45")
	t.Setenv(features.EnvTimeBasedMCKeepRecent, "3")
	c := GetTimeBasedMCConfig()
	if !c.Enabled || c.GapThresholdMinutes != 45 || c.KeepRecent != 3 {
		t.Fatalf("%+v", c)
	}
}
