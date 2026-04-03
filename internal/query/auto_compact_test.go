package query

import "testing"

func TestEffectiveContextInputWindow(t *testing.T) {
	if g := EffectiveContextInputWindow(200_000, 8192); g != 200_000-8192 {
		t.Fatalf("got %d", g)
	}
	if g := EffectiveContextInputWindow(100_000, 25_000); g != 100_000-maxOutputTokensForSummaryCap {
		t.Fatalf("cap got %d", g)
	}
	if g := EffectiveContextInputWindow(1000, 2000); g != 0 {
		t.Fatalf("clamp got %d", g)
	}
}

func TestAutoCompactThresholdTokens_pctOverride(t *testing.T) {
	base := EffectiveContextInputWindow(200_000, 8192)
	th := AutoCompactThresholdTokens(base, 0)
	want := base - AutocompactBufferTokens
	if th != want {
		t.Fatalf("no override: got %d want %d", th, want)
	}
	th50 := AutoCompactThresholdTokens(base, 50)
	half := int(float64(base) * 0.5)
	if th50 != half {
		t.Fatalf("50%%: got %d want %d", th50, half)
	}
}

func TestCalculateTokenWarningState(t *testing.T) {
	// thresholdForPercent 100000, tokenUsage 70000 -> 30% left
	st := CalculateTokenWarningState(70_000, 100_000, 100_000, 87_000, true, 0)
	if st.PercentLeft != 30 {
		t.Fatalf("PercentLeft %d", st.PercentLeft)
	}
	if st.IsAboveAutoCompactThreshold {
		t.Fatal("70000 < 87000 autocompact threshold")
	}
	stAbove := CalculateTokenWarningState(90_000, 100_000, 100_000, 87_000, true, 0)
	if !stAbove.IsAboveAutoCompactThreshold {
		t.Fatal("expected above autocompact threshold")
	}
	st2 := CalculateTokenWarningState(90_000, 100_000, 100_000, 87_000, false, 0)
	if st2.IsAboveAutoCompactThreshold {
		t.Fatal("auto off should not set autocompact threshold flag")
	}
}
