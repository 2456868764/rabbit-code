package query

import (
	"errors"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestBlockingLimitPreCheckApplies_sessionMemoryFork(t *testing.T) {
	if BlockingLimitPreCheckApplies(QuerySourceSessionMemory, false) {
		t.Fatal("session_memory fork should skip blocking limit")
	}
	if !BlockingLimitPreCheckApplies("", false) {
		t.Fatal("main thread should apply when gates pass")
	}
	if BlockingLimitPreCheckApplies("", true) {
		t.Fatal("post-compact continuation should skip")
	}
}

func TestCheckBlockingLimitPreAssistant_overrideLow(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	long, err := InitialUserMessagesJSON(strings.Repeat("z", 800))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckBlockingLimitPreAssistant("m", 1024, 50_000, long, 0, "", false); err == nil {
		t.Fatal("expected ErrBlockingLimit")
	} else if !errors.Is(err, ErrBlockingLimit) {
		t.Fatalf("got %v", err)
	}
}

func TestCheckBlockingLimitPreAssistant_reactiveAndAutoSkips(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "1")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	long, err := InitialUserMessagesJSON(strings.Repeat("z", 800))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckBlockingLimitPreAssistant("m", 1024, 50_000, long, 0, "", false); err != nil {
		t.Fatalf("reactive+auto should skip synthetic blocking: %v", err)
	}
}
