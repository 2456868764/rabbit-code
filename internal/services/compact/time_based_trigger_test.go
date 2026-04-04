package compact

import (
	"strconv"
	"testing"
	"time"

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

func TestEvaluateTimeBasedTrigger_nilWhenDisabled(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "")
	ms := []TimeBasedCCMessage{{Type: "assistant", Timestamp: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)}}
	if EvaluateTimeBasedTrigger(ms, "repl_main_thread", time.Now()) != nil {
		t.Fatal()
	}
}

func TestEvaluateTimeBasedTrigger_requiresQuerySource(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "1")
	ms := []TimeBasedCCMessage{{Type: "assistant", Timestamp: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)}}
	if EvaluateTimeBasedTrigger(ms, "", time.Now()) != nil {
		t.Fatal()
	}
}

func TestEvaluateTimeBasedTrigger_firesWhenGapExceeded(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_GAP_MINUTES", "30")
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	last := now.Add(-90 * time.Minute)
	ms := []TimeBasedCCMessage{
		{Type: "user", Timestamp: ""},
		{Type: "assistant", Timestamp: last.Format(time.RFC3339)},
	}
	ev := EvaluateTimeBasedTrigger(ms, "repl_main_thread", now)
	if ev == nil || ev.GapMinutes < 89 || ev.GapMinutes > 91 {
		t.Fatalf("%+v", ev)
	}
}

func TestEvaluateTimeBasedTrigger_unixSeconds(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_GAP_MINUTES", "1")
	now := time.Unix(1_700_000_000, 0).UTC()
	lastSec := now.Add(-120 * time.Minute).Unix()
	ms := []TimeBasedCCMessage{{Type: "assistant", Timestamp: strconv.FormatInt(lastSec, 10)}}
	ev := EvaluateTimeBasedTrigger(ms, "repl_main_thread", now)
	if ev == nil {
		t.Fatal()
	}
}

func TestEvaluateTimeBasedTriggerFromWallClock(t *testing.T) {
	t.Setenv("RABBIT_CODE_TIME_BASED_MICROCOMPACT", "1")
	t.Setenv("RABBIT_CODE_TIME_BASED_MC_GAP_MINUTES", "30")
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	last := now.Add(-90 * time.Minute)
	if EvaluateTimeBasedTriggerFromWallClock("", now, last) != nil {
		t.Fatal("empty querySource")
	}
	if EvaluateTimeBasedTriggerFromWallClock("repl_main_thread", now, time.Time{}) != nil {
		t.Fatal("zero lastAssistant")
	}
	ev := EvaluateTimeBasedTriggerFromWallClock("repl_main_thread", now, last)
	if ev == nil || ev.GapMinutes < 89 || ev.GapMinutes > 91 {
		t.Fatalf("%+v", ev)
	}
}
