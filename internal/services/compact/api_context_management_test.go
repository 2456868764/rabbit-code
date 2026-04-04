package compact

import (
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

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
