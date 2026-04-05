package features

import "testing"

func TestGetToolSearchMode_emptyEnvIsTST(t *testing.T) {
	t.Setenv("ENABLE_TOOL_SEARCH", "")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	if GetToolSearchMode() != ToolSearchModeTST {
		t.Fatalf("got %q", GetToolSearchMode())
	}
}

func TestGetToolSearchMode_explicitFalse(t *testing.T) {
	t.Setenv("ENABLE_TOOL_SEARCH", "false")
	if GetToolSearchMode() != ToolSearchModeStandard {
		t.Fatalf("got %q", GetToolSearchMode())
	}
}

func TestGetToolSearchMode_auto100(t *testing.T) {
	t.Setenv("ENABLE_TOOL_SEARCH", "auto:100")
	if GetToolSearchMode() != ToolSearchModeStandard {
		t.Fatalf("got %q", GetToolSearchMode())
	}
}
