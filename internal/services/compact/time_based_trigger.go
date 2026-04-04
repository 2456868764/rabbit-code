package compact

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// TimeBasedMCClearedMessage mirrors microCompact.ts TIME_BASED_MC_CLEARED_MESSAGE.
const TimeBasedMCClearedMessage = "[Old tool result content cleared]"

// TimeBasedCCMessage is the minimal upstream Message shape for evaluateTimeBasedTrigger (type + timestamp).
type TimeBasedCCMessage struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

// TimeBasedTriggerEval mirrors evaluateTimeBasedTrigger non-null return (gapMinutes + config snapshot).
type TimeBasedTriggerEval struct {
	GapMinutes float64
	Config     TimeBasedMCConfig
}

// EvaluateTimeBasedTrigger mirrors microCompact.ts evaluateTimeBasedTrigger.
// now is usually time.Now(); injectable for tests.
func EvaluateTimeBasedTrigger(messages []TimeBasedCCMessage, querySource string, now time.Time) *TimeBasedTriggerEval {
	cfg := GetTimeBasedMCConfig()
	qs := strings.TrimSpace(querySource)
	if !cfg.Enabled || qs == "" || !IsMainThreadQuerySource(qs) {
		return nil
	}
	var last *TimeBasedCCMessage
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == "assistant" {
			last = &messages[i]
			break
		}
	}
	if last == nil {
		return nil
	}
	ts, err := parseAssistantTimestamp(last.Timestamp)
	if err != nil {
		return nil
	}
	gapMinutes := now.Sub(ts).Minutes()
	if math.IsNaN(gapMinutes) || math.IsInf(gapMinutes, 0) {
		return nil
	}
	if gapMinutes < float64(cfg.GapThresholdMinutes) {
		return nil
	}
	return &TimeBasedTriggerEval{GapMinutes: gapMinutes, Config: cfg}
}

func parseAssistantTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	if len(s) == 10 || len(s) == 13 {
		if allASCIIDigits(s) {
			sec, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			if len(s) == 10 {
				return time.Unix(sec, 0).UTC(), nil
			}
			return time.UnixMilli(sec).UTC(), nil
		}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func allASCIIDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
