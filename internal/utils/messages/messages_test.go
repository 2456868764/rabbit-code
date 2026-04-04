package messages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/2456868764/rabbit-code/internal/types"
)

func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	dir := filepath.Dir(file)
	path := filepath.Join(dir, "testdata", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

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

func TestGolden_transcript_v1_minimal(t *testing.T) {
	b := readGolden(t, "transcript_v1_minimal.json")
	tr, err := ParseTranscriptJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if tr.TranscriptVersion != 1 || len(tr.Messages) != 2 {
		t.Fatalf("%+v", tr)
	}
	if tr.Messages[0].Role != "user" || tr.Messages[1].Role != "assistant" {
		t.Fatal(tr.Messages)
	}
	out, err := CanonicalJSON(tr)
	if err != nil {
		t.Fatal(err)
	}
	tr2, err := ParseTranscriptJSON(out)
	if err != nil {
		t.Fatal(err)
	}
	h1, _ := SHA256Hex(tr)
	h2, _ := SHA256Hex(tr2)
	if h1 != h2 {
		t.Fatalf("hash drift %s vs %s", h1, h2)
	}
}

func TestGolden_transcript_tool_chain_strict(t *testing.T) {
	b := readGolden(t, "transcript_tool_chain.json")
	tr, err := ParseTranscriptJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateToolPairing(tr.Messages, true); err != nil {
		t.Fatal(err)
	}
}

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

func TestE2E_TranscriptRoundTripHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")

	tr := &Transcript{
		TranscriptVersion: CurrentTranscriptVersion,
		Messages: []types.Message{
			{Role: types.RoleUser, Content: []types.ContentPiece{{Type: types.BlockTypeText, Text: "e2e"}}},
			{Role: types.RoleAssistant, Content: []types.ContentPiece{{Type: types.BlockTypeText, Text: "ok"}}},
		},
	}
	h1, err := SHA256Hex(tr)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteTranscriptFile(path, tr); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTranscriptFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("count %d", len(got.Messages))
	}
	if got.Messages[0].Role != types.RoleUser {
		t.Fatal(got.Messages[0].Role)
	}
	h2, err := SHA256Hex(got)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("hash mismatch %s vs %s", h1, h2)
	}
}

func TestVerifyFileRefs_ok(t *testing.T) {
	f := filepath.Join(t.TempDir(), "blob.bin")
	payload := []byte("payload-bytes")
	if err := os.WriteFile(f, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	msgs := []types.Message{
		{
			Role: types.RoleUser,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeFileRef, Ref: f, Sha256: hex.EncodeToString(sum[:])},
			},
		},
	}
	if err := VerifyFileRefs(msgs); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFileRefs_badHash(t *testing.T) {
	f := filepath.Join(t.TempDir(), "x")
	_ = os.WriteFile(f, []byte("a"), 0o600)
	msgs := []types.Message{
		{
			Role: types.RoleUser,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeFileRef, Ref: f, Sha256: "00" + "00"},
			},
		},
	}
	if err := VerifyFileRefs(msgs); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeriveShortMessageId_deterministic(t *testing.T) {
	u := "550e8400-e29b-41d4-a716-446655440000"
	a := DeriveShortMessageId(u)
	b := DeriveShortMessageId(u)
	if a == "" || a != b {
		t.Fatalf("got %q %q", a, b)
	}
	if len(a) > 6 {
		t.Fatalf("len %d", len(a))
	}
}

func TestReorderAttachmentsForAPIGeneric(t *testing.T) {
	// Bottom attachment bubbles to after assistant when scanning TS algorithm.
	msgs := []map[string]any{
		{"type": "user", "message": map[string]any{"content": "hi"}},
		{"type": "assistant", "message": map[string]any{"content": []any{map[string]any{"type": "text", "text": "a"}}}},
		{"type": "attachment", "attachment": map[string]any{"k": 1}},
	}
	out := ReorderAttachmentsForAPIGeneric(msgs)
	if len(out) != 3 {
		t.Fatalf("len %d", len(out))
	}
	// TS: trailing attachment bubbles up until assistant; reverse pass yields user, assistant, attachment.
	if out[0]["type"] != "user" || out[1]["type"] != "assistant" || out[2]["type"] != "attachment" {
		t.Fatalf("order %+v", []any{out[0]["type"], out[1]["type"], out[2]["type"]})
	}
}

func TestNormalizeMessagesForAPIGeneric_mergeUsersStripCaller(t *testing.T) {
	msgs := []map[string]any{
		{"type": "user", "message": map[string]any{"content": "a"}},
		{"type": "user", "message": map[string]any{"content": "b"}},
		{
			"type": "assistant",
			"message": map[string]any{
				"id": "m1",
				"content": []any{
					map[string]any{"type": "tool_use", "id": "t1", "name": "Read", "input": map[string]any{}, "caller": "x"},
				},
			},
		},
	}
	out, err := NormalizeMessagesForAPIGeneric(msgs, NormalizeMessagesForAPIConfig{ToolSearchEnabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 got %d", len(out))
	}
	u := out[0]
	uc := u["message"].(map[string]any)["content"].([]any)
	if len(uc) != 1 {
		t.Fatalf("merged user content %+v", uc)
	}
	tb := uc[0].(map[string]any)
	if tb["text"] != "a\nb" {
		t.Fatalf("text %q", tb["text"])
	}
	tu := out[1]["message"].(map[string]any)["content"].([]any)[0].(map[string]any)
	if _, has := tu["caller"]; has {
		t.Fatal("caller should be stripped")
	}
}

func TestNormalizeMessagesForAPIGeneric_stripMetaAfterSyntheticPdfError(t *testing.T) {
	t.Setenv("RABBIT_NON_INTERACTIVE", "")
	errText := "The PDF file was not valid. Double press esc to go back and try again with a different file."
	msgs := []map[string]any{
		{
			"type": "user", "uuid": "u-meta", "isMeta": true,
			"message": map[string]any{
				"content": []any{
					map[string]any{"type": "document", "title": "t"},
					map[string]any{"type": "text", "text": "keep"},
				},
			},
		},
		{
			"type": "assistant", "isApiErrorMessage": true,
			"message": map[string]any{
				"model": SyntheticModel,
				"content": []any{
					map[string]any{"type": "text", "text": errText},
				},
			},
		},
	}
	out, err := NormalizeMessagesForAPIGeneric(msgs, NormalizeMessagesForAPIConfig{ToolSearchEnabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 message got %d", len(out))
	}
	arr := out[0]["message"].(map[string]any)["content"].([]any)
	if len(arr) != 1 {
		t.Fatalf("want 1 block after document strip, got %d", len(arr))
	}
	tb := arr[0].(map[string]any)
	if tb["type"] != "text" || tb["text"] != "keep" {
		t.Fatalf("got %+v", tb)
	}
}

func TestMergeUserMessagesMap_snipClearsMetaWhenMixed(t *testing.T) {
	t.Setenv("RABBIT_HISTORY_SNIP", "1")
	t.Setenv("RABBIT_SNIP_RUNTIME_ENABLED", "1")
	a := map[string]any{
		"type": "user", "uuid": "a", "isMeta": true,
		"message": map[string]any{"content": "m"},
	}
	b := map[string]any{
		"type": "user", "uuid": "b",
		"message": map[string]any{"content": "u"},
	}
	out := MergeUserMessagesMap(a, b)
	if _, has := out["isMeta"]; has {
		t.Fatalf("expected isMeta cleared when merging meta+non-meta under snip, got %+v", out)
	}
}

func TestNormalizeToolInputForAPIMap_exitPlanStripsPlan(t *testing.T) {
	in := map[string]any{"plan": "x", "ok": true}
	out := NormalizeToolInputForAPIMap(ToolNameExitPlanModeV2, in)
	if _, ok := out["plan"]; ok {
		t.Fatalf("plan should be stripped: %+v", out)
	}
	if out["ok"] != true {
		t.Fatal(out)
	}
}

func TestNormalizeLegacyToolName_taskAlias(t *testing.T) {
	if NormalizeLegacyToolName("Task") != ToolNameAgent {
		t.Fatalf("got %q", NormalizeLegacyToolName("Task"))
	}
}

func TestValidateImagesForAPIMap_rejectsOversizedBase64(t *testing.T) {
	big := strings.Repeat("x", 5*1024*1024+1)
	msgs := []map[string]any{{
		"type": "user",
		"message": map[string]any{
			"content": []any{map[string]any{
				"type": "image",
				"source": map[string]any{
					"type": "base64",
					"data": big,
				},
			}},
		},
	}}
	err := ValidateImagesForAPIMap(msgs)
	if err == nil {
		t.Fatal("expected ImageSizeError")
	}
}

func TestIsSyntheticMessageMap(t *testing.T) {
	m := map[string]any{
		"type": "user",
		"message": map[string]any{
			"content": []any{map[string]any{"type": "text", "text": CancelMessage}},
		},
	}
	if !IsSyntheticMessageMap(m) {
		t.Fatal("expected synthetic")
	}
}

func TestNotebookMapCellsToToolResultBlocks_mergesAdjacentText(t *testing.T) {
	fc := map[string]any{
		"type": "notebook",
		"file": map[string]any{
			"cells": []any{
				map[string]any{"cell_type": "markdown", "cell_id": "c0", "source": "alpha"},
				map[string]any{"cell_type": "markdown", "cell_id": "c1", "source": "beta"},
			},
		},
	}
	blocks := notebookMapCellsToToolResultBlocks(fc)
	if len(blocks) != 1 {
		t.Fatalf("want 1 merged text block, got %d %#v", len(blocks), blocks)
	}
	txt, _ := blocks[0]["text"].(string)
	if !strings.Contains(txt, "alpha") || !strings.Contains(txt, "beta") {
		t.Fatalf("unexpected merged text: %q", txt)
	}
	if !strings.Contains(txt, "\n") {
		t.Fatalf("expected newline between merged cell texts: %q", txt)
	}
}

func TestNotebookReadToolResultMessage_withImageNoResultPrefix(t *testing.T) {
	blocks := []map[string]any{
		{"type": "text", "text": "hi"},
		{"type": "image", "source": map[string]any{"type": "base64", "data": "QQ==", "media_type": "image/png"}},
	}
	msg := notebookReadToolResultMessage(ToolNameRead, blocks)
	mm, _ := msg["message"].(map[string]any)
	content := mm["content"]
	arr, ok := content.([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("want content []any len 2, got %#v", content)
	}
}

func TestFormatDiagnosticsSummary_lineAndSeverity(t *testing.T) {
	files := []any{map[string]any{
		"uri": "file:///proj/src/a.go",
		"diagnostics": []any{map[string]any{
			"severity": "Error",
			"message":  "oops",
			"code":     "E1",
			"source":   "compiler",
			"range": map[string]any{
				"start": map[string]any{"line": 4.0, "character": 1.0},
			},
		}},
	}}
	s := FormatDiagnosticsSummary(files)
	if !strings.Contains(s, "a.go:") || !strings.Contains(s, "[Line 5:2]") || !strings.Contains(s, "oops") {
		t.Fatalf("unexpected summary: %q", s)
	}
}

func TestMemoryHeader_freshVsStale(t *testing.T) {
	now := time.Now().UnixMilli()
	h := MemoryHeader("/x.md", now)
	if !strings.Contains(h, "today") && !strings.Contains(h, "Memory (saved") {
		t.Fatalf("expected age phrase: %q", h)
	}
	old := now - 5*86400000
	h2 := MemoryHeader("/y.md", old)
	if !strings.Contains(h2, "5 days old") {
		t.Fatalf("expected staleness: %q", h2)
	}
}

func TestFileReadTextToolResultString_addsLineNumbers(t *testing.T) {
	fc := map[string]any{
		"type": "text",
		"file": map[string]any{
			"content":   "a\nb",
			"startLine": float64(1),
		},
	}
	s := fileReadTextToolResultString(fc)
	if !strings.Contains(s, "\u2192") && !strings.Contains(s, "\t") {
		t.Fatalf("expected numbered lines, got %q", s)
	}
}

func TestNotebookReadToolResultMessage_textOnlyUsesJSONArrayWrapper(t *testing.T) {
	blocks := []map[string]any{{"type": "text", "text": "only"}}
	msg := notebookReadToolResultMessage(ToolNameRead, blocks)
	mm, _ := msg["message"].(map[string]any)
	s, ok := mm["content"].(string)
	if !ok || !strings.HasPrefix(s, "Result of calling the "+ToolNameRead+" tool:\n") {
		t.Fatalf("want string tool result prefix, got %#v", mm["content"])
	}
	if !strings.Contains(s, `"type":"text"`) {
		t.Fatalf("expected json-serialized blocks in body: %q", s)
	}
}

func TestDefaultFormatTeammateMailboxMessagesForAPI(t *testing.T) {
	s := DefaultFormatTeammateMailboxMessagesForAPI([]any{
		map[string]any{"from": "alice", "text": "hello", "color": "blue", "summary": "hi"},
	})
	if !strings.Contains(s, `<teammate-message teammate_id="alice" color="blue" summary="hi">`) {
		t.Fatalf("unexpected xml: %q", s)
	}
	if !strings.Contains(s, "hello") {
		t.Fatalf("missing body: %q", s)
	}
}

func TestBashAttachmentToolResultContentString_isImageInvalidFallsBackToText(t *testing.T) {
	s := BashAttachmentToolResultContentString(map[string]any{
		"bash": map[string]any{
			"stdout":  "AAA",
			"isImage": true,
		},
	})
	if s != "AAA" {
		t.Fatalf("TS falls through to text when data URI parse fails; got %q", s)
	}
}

func TestBashAttachmentToolResultContentString_isImageDataURIPlaceholder(t *testing.T) {
	uri := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	s := BashAttachmentToolResultContentString(map[string]any{
		"bash": map[string]any{"stdout": uri, "isImage": true},
	})
	if !strings.Contains(s, "Image output") {
		t.Fatalf("got %q", s)
	}
}

func TestBashToolResultMetaMessage_imageBlocks(t *testing.T) {
	uri := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	m := BashToolResultMetaMessage(ToolNameBash, map[string]any{
		"bash": map[string]any{"stdout": uri, "isImage": true},
	})
	inner, _ := m["message"].(map[string]any)
	arr, ok := inner["content"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("expected one content block, got %#v", inner["content"])
	}
	b0, _ := arr[0].(map[string]any)
	if b0["type"] != "image" {
		t.Fatalf("got %#v", b0)
	}
}

func TestNormalizeAttachmentForAPI_directory_bashImageBlocks(t *testing.T) {
	uri := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	msgs, err := NormalizeAttachmentForAPI(map[string]any{
		"type":    "directory",
		"path":    "/tmp",
		"content": uri,
		"bash":    map[string]any{"isImage": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) < 1 {
		t.Fatal("no messages")
	}
	// Wrapped in system-reminder user messages; find image block in chain.
	var found bool
	for _, msg := range msgs {
		inner, _ := msg["message"].(map[string]any)
		if arr, ok := inner["content"].([]any); ok {
			for _, it := range arr {
				b, ok := it.(map[string]any)
				if ok && b["type"] == "image" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatalf("no image in %#v", msgs)
	}
}

func TestOutputStyleDisplayName_fromEnvJSON(t *testing.T) {
	t.Setenv("RABBIT_OUTPUT_STYLE_NAMES_JSON", `{"Acme":"Acme Mode"}`)
	if n := outputStyleDisplayName("Acme"); n != "Acme Mode" {
		t.Fatalf("got %q", n)
	}
}

func TestBashBackgroundTaskOutputPath_fromEnv(t *testing.T) {
	t.Setenv("RABBIT_TASK_OUTPUT_DIR", "/var/task-out")
	s := BashAttachmentToolResultContentString(map[string]any{
		"bash": map[string]any{
			"stdout":                    "ok",
			"backgroundTaskId":          "task-abc",
			"assistantAutoBackgrounded": true,
		},
	})
	if !strings.Contains(s, "/var/task-out/task-abc.output") {
		t.Fatalf("expected derived path in message, got %q", s)
	}
}

func TestBashAttachmentToolResultContentString(t *testing.T) {
	s := BashAttachmentToolResultContentString(map[string]any{
		"content": "  \n\nhello",
		"stderr":  "warn",
	})
	if !strings.Contains(s, "hello") || !strings.Contains(s, "warn") {
		t.Fatalf("got %q", s)
	}
	s2 := BashAttachmentToolResultContentString(map[string]any{
		"bash": map[string]any{
			"stdout":              "out",
			"persistedOutputPath": "/tmp/x.txt",
			"persistedOutputSize": float64(99999),
		},
	})
	if !strings.Contains(s2, "<persisted-output>") || !strings.Contains(s2, "/tmp/x.txt") {
		t.Fatalf("persisted: %q", s2)
	}
}

func TestFileReadTextToolResultString_skipsDoubleLineNumbers(t *testing.T) {
	fc := map[string]any{
		"type": "text",
		"file": map[string]any{
			"content":   "1\u2192already-numbered",
			"startLine": float64(1),
		},
	}
	out := fileReadTextToolResultString(fc)
	if !strings.Contains(out, "already-numbered") {
		t.Fatalf("missing body: %q", out)
	}
	// If we wrongly ran addLineNumbers on top, first line would become "1<TAB>1→..." in compact mode.
	if strings.Contains(out, "\t1\u2192") {
		t.Fatalf("double line-number pass: %q", out)
	}
}

func TestFormatAttachmentNumberForTemplate(t *testing.T) {
	if formatAttachmentNumberForTemplate(3) != "3" {
		t.Fatal()
	}
	if formatAttachmentNumberForTemplate(3.5) != "3.5" {
		t.Fatal(formatAttachmentNumberForTemplate(3.5))
	}
}

func TestNormalizeLegacyToolName_featureKairos(t *testing.T) {
	t.Setenv("RABBIT_FEATURE_KAIROS", "1")
	if NormalizeLegacyToolName("Brief") != "SendUserMessage" {
		t.Fatal(NormalizeLegacyToolName("Brief"))
	}
}

func TestNormalizeAttachmentForAPI_teammateMailbox_defaultFormat(t *testing.T) {
	t.Setenv("RABBIT_AGENT_SWARMS", "1")
	msgs, err := NormalizeAttachmentForAPI(map[string]any{
		"type": "teammate_mailbox",
		"messages": []any{
			map[string]any{"from": "bob", "text": "ping"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d msgs", len(msgs))
	}
	mm, _ := msgs[0]["message"].(map[string]any)
	body, _ := mm["content"].(string)
	if !strings.Contains(body, `teammate_id="bob"`) || !strings.Contains(body, "ping") {
		t.Fatalf("unexpected content: %q", body)
	}
}

func TestMergeUserContentBlocks_stringToolResultSmoosh(t *testing.T) {
	t.Setenv("RABBIT_TENGU_CHAIR_SERMON", "")
	a := []map[string]any{{
		"type":    "tool_result",
		"content": "base",
	}}
	b := []map[string]any{{"type": "text", "text": "extra"}}
	out := MergeUserContentBlocks(a, b)
	if len(out) != 1 {
		t.Fatalf("len=%d %#v", len(out), out)
	}
	s, _ := out[0]["content"].(string)
	if !strings.Contains(s, "base") || !strings.Contains(s, "extra") {
		t.Fatalf("got %q", s)
	}
}

func TestSanitizeErrorToolResultContentGeneric_stripsNonText(t *testing.T) {
	msgs := []map[string]any{{
		"type": "user",
		"message": map[string]any{
			"content": []any{map[string]any{
				"type":     "tool_result",
				"is_error": true,
				"content": []any{
					map[string]any{"type": "text", "text": "err"},
					map[string]any{"type": "image", "source": map[string]any{"type": "base64", "data": "abc", "media_type": "image/png"}},
				},
			}},
		},
	}}
	out := sanitizeErrorToolResultContentGeneric(msgs)
	inner, _ := out[0]["message"].(map[string]any)
	arr, _ := inner["content"].([]any)
	tr, _ := arr[0].(map[string]any)
	trc, _ := tr["content"].([]any)
	if len(trc) != 1 {
		t.Fatalf("got %#v", trc)
	}
	tb, _ := trc[0].(map[string]any)
	if tb["type"] != "text" || tb["text"] != "err" {
		t.Fatalf("got %#v", tb)
	}
}
