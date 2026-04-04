package compact

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
	"github.com/2456868764/rabbit-code/internal/tools/webfetchtool"
	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
)

func TestMicroCompactRequested(t *testing.T) {
	_ = os.Unsetenv("RABBIT_CODE_CACHED_MICROCOMPACT")
	if MicroCompactRequested() {
		t.Fatal()
	}
	t.Setenv("RABBIT_CODE_CACHED_MICROCOMPACT", "1")
	if !MicroCompactRequested() {
		t.Fatal()
	}
}

func TestPromptCacheBreakActive(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	if !PromptCacheBreakActive() {
		t.Fatal()
	}
}

func TestGetAPIContextManagement_nilWhenEmpty(t *testing.T) {
	t.Setenv(features.EnvUserType, "")
	t.Setenv(features.EnvUserTypeRabbit, "")
	if g := GetAPIContextManagement(APIContextManagementOptions{}); g != nil {
		t.Fatalf("%+v", g)
	}
}

func TestGetAPIContextManagement_thinkingOnly(t *testing.T) {
	t.Setenv(features.EnvUserType, "")
	cfg := GetAPIContextManagement(APIContextManagementOptions{HasThinking: true})
	if cfg == nil || len(cfg.Edits) != 1 {
		t.Fatalf("%+v", cfg)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(cfg.Edits[0], &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "clear_thinking_20251015" {
		t.Fatalf("%v", m)
	}
}

func TestGetAPIContextManagement_antClearResults(t *testing.T) {
	t.Setenv(features.EnvUserType, "ant")
	t.Setenv(features.EnvUseAPIClearToolResults, "1")
	t.Setenv(features.EnvUseAPIClearToolUses, "")
	t.Setenv(features.EnvAPIMaxInputTokens, "")
	t.Setenv(features.EnvAPITargetInputTokens, "")
	cfg := GetAPIContextManagement(APIContextManagementOptions{})
	if cfg == nil || len(cfg.Edits) != 1 {
		t.Fatalf("%+v", cfg)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(cfg.Edits[0], &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "clear_tool_uses_20250919" {
		t.Fatal()
	}
	cti := m["clear_tool_inputs"].([]interface{})
	if len(cti) < 5 {
		t.Fatalf("clear_tool_inputs len %d", len(cti))
	}
}

func TestGetAPIContextManagement_nonAntSkipsToolStrategies(t *testing.T) {
	t.Setenv(features.EnvUserType, "external")
	t.Setenv(features.EnvUseAPIClearToolResults, "1")
	cfg := GetAPIContextManagement(APIContextManagementOptions{})
	if cfg != nil {
		t.Fatalf("expected nil got %+v", cfg)
	}
}

func TestAPIClearToolInputs_orderMatchesMicrocompactTS(t *testing.T) {
	got := toolsClearableResults()
	// TS: ...SHELL_TOOL_NAMES, GLOB, GREP, FILE_READ, WEB_FETCH, WEB_SEARCH
	if len(got) < 6 {
		t.Fatal(got)
	}
	n := len(got)
	if got[n-5] != globtool.GlobToolName || got[n-4] != greptool.GrepToolName ||
		got[n-3] != filereadtool.FileReadToolName || got[n-2] != webfetchtool.WebFetchToolName ||
		got[n-1] != websearchtool.WebSearchToolName {
		t.Fatalf("tail order: %v", got)
	}
}

func TestDefaultAPITokenConstants_matchTS(t *testing.T) {
	if DefaultAPIMaxInputTokens != 180_000 || DefaultAPITargetInputTokens != 40_000 {
		t.Fatal()
	}
}
