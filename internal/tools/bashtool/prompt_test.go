package bashtool_test

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestGetSimplePrompt_containsLeadAndInstructions(t *testing.T) {
	t.Setenv(features.EnvMonitorTool, "")
	t.Setenv(features.EnvEmbeddedSearchTools, "")
	p := bashtool.GetSimplePrompt()
	if !strings.Contains(p, bashtool.PromptLead) || !strings.Contains(p, "# Instructions") {
		t.Fatalf("len=%d head=%.80q", len(p), p)
	}
	if strings.Contains(p, "Monitor tool to stream") {
		t.Fatal("monitor bullets should be off without RABBIT_MONITOR_TOOL")
	}
}

func TestGetSimplePrompt_monitorBullets(t *testing.T) {
	t.Setenv(features.EnvMonitorTool, "1")
	p := bashtool.GetSimplePrompt()
	if !strings.Contains(p, "Monitor tool to stream") || !strings.Contains(p, "sleep N` as the first command") {
		t.Fatal(p)
	}
}
