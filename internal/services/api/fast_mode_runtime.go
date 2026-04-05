package anthropic

import (
	"sync"
	"time"
)

// FastMode cooldown mirrors fastMode.ts runtime state (operational fast speed vs cooldown after overload/rate-limit).

var fastModeRuntimeMu sync.Mutex
var fastModeCooldownUntilMs int64 // unix milliseconds; 0 = active

// IsFastModeCooldown returns true while in cooldown window (getFastModeRuntimeState).
func IsFastModeCooldown() bool {
	fastModeRuntimeMu.Lock()
	defer fastModeRuntimeMu.Unlock()
	now := time.Now().UnixMilli()
	if fastModeCooldownUntilMs > 0 && now >= fastModeCooldownUntilMs {
		fastModeCooldownUntilMs = 0
	}
	return fastModeCooldownUntilMs > 0 && now < fastModeCooldownUntilMs
}

// TriggerFastModeCooldown sets cooldown until resetAt (unix ms), e.g. from Retry-After (triggerFastModeCooldown).
func TriggerFastModeCooldown(resetAtUnixMs int64, _ string) {
	if resetAtUnixMs <= 0 {
		return
	}
	fastModeRuntimeMu.Lock()
	fastModeCooldownUntilMs = resetAtUnixMs
	fastModeRuntimeMu.Unlock()
}

// ClearFastModeCooldown clears operational cooldown (clearFastModeCooldown).
func ClearFastModeCooldown() {
	fastModeRuntimeMu.Lock()
	fastModeCooldownUntilMs = 0
	fastModeRuntimeMu.Unlock()
}
