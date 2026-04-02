package anthropic

import (
	"math"
	"testing"
)

func TestMinBackoffMS_FastVsDefault(t *testing.T) {
	if minBackoffMS(false) != BaseDelayMS || minBackoffMS(true) != FastBaseDelayMS {
		t.Fatalf("got %d %d", minBackoffMS(false), minBackoffMS(true))
	}
}

func TestBackoffDuration_FastLowerFloor(t *testing.T) {
	// Statistical: fast policy floor is FastBaseDelayMS; default is BaseDelayMS. Compare many samples.
	const samples = 200
	var fastSum, defSum float64
	for i := 0; i < samples; i++ {
		fastSum += BackoffDuration(0, Policy{FastRetry: true}).Seconds()
		defSum += BackoffDuration(0, Policy{FastRetry: false}).Seconds()
	}
	fastMean := fastSum / samples
	defMean := defSum / samples
	if fastMean >= defMean {
		t.Fatalf("fastMean=%v defMean=%v", fastMean, defMean)
	}
	// Expect means near 0.1s vs 0.5s (± jitter); allow generous margin.
	if fastMean > 0.2 || defMean < 0.3 {
		t.Fatalf("fastMean=%v defMean=%v", fastMean, defMean)
	}
}

func TestBackoff_UnattendedCapUsesAttempt(t *testing.T) {
	pol := Policy{Unattended: true, FastRetry: false}
	// Large attempt index would exceed 5min cap without cap logic.
	ms := backoff(20, pol.Unattended, pol.FastRetry).Seconds() * 1000
	if ms > 5*60*1000*(1.0+0.15) { // cap + max jitter
		t.Fatalf("backoff too large: %g ms", ms)
	}
	if ms < 5*60*1000*(1.0-0.15) {
		// Should sit near cap when attempt is huge
		t.Fatalf("expected capped backoff, got %g ms", ms)
	}
}

func TestBackoff_AttemptZeroMatchesFloorOrder(t *testing.T) {
	// attempt 0 is unused in DoRequest sleeps; still document floor ordering.
	f := backoff(0, false, true).Seconds()
	d := backoff(0, false, false).Seconds()
	if f >= d || f < float64(FastBaseDelayMS)/1000*0.85 {
		t.Fatalf("f=%v d=%v", f, d)
	}
	if math.Abs(d-float64(BaseDelayMS)/1000) > float64(BaseDelayMS)/1000*0.15 {
		t.Fatalf("d=%v", d)
	}
}
