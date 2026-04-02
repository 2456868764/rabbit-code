package messages

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/types"
)

func TestStripHistorySnipPieces(t *testing.T) {
	msgs := []types.Message{
		{Role: types.RoleUser, Content: []types.ContentPiece{
			{Type: types.BlockTypeHistorySnip, SnipID: "a", SnipEdge: "start"},
			{Type: types.BlockTypeText, Text: "keep"},
		}},
		{Role: types.RoleUser, Content: []types.ContentPiece{
			{Type: types.BlockTypeHistorySnip},
		}},
	}
	out := StripHistorySnipPieces(msgs)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if len(out[0].Content) != 1 || out[0].Content[0].Text != "keep" {
		t.Fatalf("%+v", out[0].Content)
	}
}
