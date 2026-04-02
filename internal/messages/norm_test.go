package messages

import (
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/types"
)

func TestNormalizeForAPI_stripsInternal(t *testing.T) {
	b := readGolden(t, "transcript_with_internal.json")
	tr, err := ParseTranscriptJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	out := NormalizeForAPI(tr.Messages, DefaultNormalizeAPI())
	if len(out) != 2 {
		t.Fatalf("want 2 api messages got %d: %+v", len(out), out)
	}
	if out[0].Role != types.RoleUser || len(out[0].Content) != 1 || out[0].Content[0].Text != "visible" {
		t.Fatalf("%+v", out[0])
	}
	if out[1].Role != types.RoleAssistant || len(out[1].Content) != 1 {
		t.Fatalf("%+v", out[1])
	}
	if out[1].Content[0].Type != types.BlockTypeText || out[1].Content[0].Text != "from connector" {
		t.Fatalf("connector_text should become text: %+v", out[1].Content[0])
	}
}

func TestNormalizeForAPI_fileRefHTTPSMapsToDocument(t *testing.T) {
	msgs := []types.Message{{
		Role: types.RoleUser,
		Content: []types.ContentPiece{{
			Type:      types.BlockTypeFileRef,
			Ref:       "https://example.com/a.pdf",
			MediaType: "application/pdf",
		}},
	}}
	out := NormalizeForAPI(msgs, DefaultNormalizeAPI())
	if len(out) != 1 || len(out[0].Content) != 1 {
		t.Fatalf("got %+v", out)
	}
	d := out[0].Content[0]
	if d.Type != types.BlockTypeDocument {
		t.Fatalf("type %q", d.Type)
	}
	var src struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal(d.Source, &src); err != nil {
		t.Fatal(err)
	}
	if src.Type != "url" || src.URL != "https://example.com/a.pdf" {
		t.Fatalf("%+v", src)
	}
}

func TestNormalizeForAPI_fileRefNonURLStripped(t *testing.T) {
	msgs := []types.Message{{
		Role: types.RoleUser,
		Content: []types.ContentPiece{{
			Type: types.BlockTypeFileRef,
			Ref:  "/local/path.pdf",
		}},
	}}
	out := NormalizeForAPI(msgs, DefaultNormalizeAPI())
	if len(out) != 0 {
		t.Fatalf("expected drop, got %+v", out)
	}
}

func TestNormalizeForAPI_preservesToolChain(t *testing.T) {
	b := readGolden(t, "transcript_tool_chain.json")
	tr, err := ParseTranscriptJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	out := NormalizeForAPI(tr.Messages, DefaultNormalizeAPI())
	if len(out) != 3 {
		t.Fatalf("got %d", len(out))
	}
	if out[1].Content[0].Type != types.BlockTypeToolUse {
		t.Fatal(out[1].Content[0])
	}
	if out[2].Content[0].Type != types.BlockTypeToolResult {
		t.Fatal(out[2].Content[0])
	}
}

func TestFeatureBlockRoundTrip_JSON(t *testing.T) {
	// P3.F.1–F.7 structural presence (AC3-F* baseline)
	pieces := []types.ContentPiece{
		{Type: types.BlockTypeHistorySnip, SnipID: "x", SnipEdge: "end"},
		{Type: types.BlockTypeCompactionReminder, ReminderID: "r1", Text: "compact soon"},
		{Type: types.BlockTypeKairosQueue, QueueID: "q1"},
		{Type: types.BlockTypeKairosChannel, ChannelID: "c1", PlanID: "p1"},
		{Type: types.BlockTypeKairosBrief, BriefID: "b1"},
		{Type: types.BlockTypeUDSInbox, InboxAddress: "inbox://main"},
	}
	b, err := json.Marshal(pieces)
	if err != nil {
		t.Fatal(err)
	}
	var got []types.ContentPiece
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(pieces) {
		t.Fatal(len(got))
	}
}
