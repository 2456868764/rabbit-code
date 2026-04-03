package messages

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/types"
)

func TestValidateToolPairing_strict_missingResult(t *testing.T) {
	msgs := []types.Message{
		{
			Role: types.RoleAssistant,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeToolUse, ID: "a", Name: "n", Input: json.RawMessage(`{}`)},
			},
		},
		{
			Role: types.RoleUser,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeToolResult, ToolUseID: "wrong", Content: json.RawMessage(`"x"`)},
			},
		},
	}
	err := ValidateToolPairing(msgs, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing tool_result") {
		t.Fatal(err)
	}
}

func TestValidateToolPairing_strict_extraResult(t *testing.T) {
	msgs := []types.Message{
		{
			Role: types.RoleAssistant,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeToolUse, ID: "only", Name: "n", Input: json.RawMessage(`{}`)},
			},
		},
		{
			Role: types.RoleUser,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeToolResult, ToolUseID: "only", Content: json.RawMessage(`"ok"`)},
				{Type: types.BlockTypeToolResult, ToolUseID: "orphan", Content: json.RawMessage(`"x"`)},
			},
		},
	}
	err := ValidateToolPairing(msgs, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unexpected tool_result") {
		t.Fatal(err)
	}
}

func TestValidateToolPairing_nonStrict_trailingAssistant(t *testing.T) {
	msgs := []types.Message{
		{
			Role: types.RoleAssistant,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeToolUse, ID: "z", Name: "n", Input: json.RawMessage(`{}`)},
			},
		},
	}
	if err := ValidateToolPairing(msgs, false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateToolPairing(msgs, true); err == nil {
		t.Fatal("strict should fail")
	}
}
