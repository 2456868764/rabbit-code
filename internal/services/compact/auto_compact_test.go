package compact

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestEffectiveContextInputWindow(t *testing.T) {
	if g := EffectiveContextInputWindow(200_000, 8192); g != 200_000-8192 {
		t.Fatalf("got %d", g)
	}
	if g := EffectiveContextInputWindow(100_000, 25_000); g != 100_000-MaxOutputTokensForSummaryCap {
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

func TestMarshalAutoCompactTrackingJSON_roundTrip(t *testing.T) {
	v := 2
	orig := &AutoCompactTracking{Compacted: true, TurnCounter: 3, TurnID: "autocompact:1", ConsecutiveFailures: &v}
	data, err := MarshalAutoCompactTrackingJSON(orig)
	if err != nil {
		t.Fatal(err)
	}
	out, err := UnmarshalAutoCompactTrackingJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || !out.Compacted || out.TurnCounter != 3 || out.TurnID != "autocompact:1" ||
		out.ConsecutiveFailures == nil || *out.ConsecutiveFailures != 2 {
		t.Fatalf("%+v", out)
	}
	if _, err := UnmarshalAutoCompactTrackingJSON([]byte("  ")); err != nil {
		t.Fatal(err)
	}
}

func TestCloneAutoCompactTracking_nil(t *testing.T) {
	if CloneAutoCompactTracking(nil) != nil {
		t.Fatal()
	}
}

func TestCloneAutoCompactTracking_copiesPointerField(t *testing.T) {
	v := 3
	orig := &AutoCompactTracking{Compacted: true, TurnCounter: 2, TurnID: "t1", ConsecutiveFailures: &v}
	cp := CloneAutoCompactTracking(orig)
	if cp == nil || cp == orig {
		t.Fatal()
	}
	if *cp.ConsecutiveFailures != 3 {
		t.Fatal()
	}
	*cp.ConsecutiveFailures = 99
	if *orig.ConsecutiveFailures != 3 {
		t.Fatal("mutating clone should not affect original")
	}
}

func TestProactiveAutoCompactSuggested_gates(t *testing.T) {
	blob := []byte(strings.Repeat("a", 150_000))
	t.Run("disabled compact", func(t *testing.T) {
		t.Setenv(features.EnvDisableCompact, "1")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false")
		}
	})
	t.Run("context collapse suppresses proactive", func(t *testing.T) {
		t.Setenv(features.EnvDisableCompact, "")
		t.Setenv(features.EnvContextCollapse, "1")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false")
		}
	})
	t.Run("suppress proactive", func(t *testing.T) {
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvSuppressProactiveAutoCompact, "1")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false")
		}
	})
	t.Run("above threshold", func(t *testing.T) {
		t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
		t.Setenv(features.EnvDisableCompact, "")
		t.Setenv(features.EnvDisableAutoCompact, "")
		t.Setenv(features.EnvAutoCompact, "")
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if !ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected true when transcript exceeds autocompact threshold")
		}
	})
	t.Run("querySource session_memory blocks", func(t *testing.T) {
		t.Setenv(features.EnvDisableCompact, "")
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggestedWithSource(blob, "m", 1024, 0, 0, QuerySourceSessionMemory) {
			t.Fatal("expected false for forked session_memory")
		}
	})
	t.Run("reactive compact plus cobalt suppresses proactive", func(t *testing.T) {
		t.Setenv(features.EnvReactiveCompact, "1")
		t.Setenv(features.EnvTenguCobaltRaccoon, "1")
		t.Setenv(features.EnvContextCollapse, "")
		t.Setenv(features.EnvContextWindowTokens, "50000")
		if ProactiveAutoCompactSuggested(blob, "m", 1024, 0, 0) {
			t.Fatal("expected false when reactive+cobalt")
		}
	})
}
