package query

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestCheckTokenBudget_skipsWithAgentID(t *testing.T) {
	tr := NewBudgetTracker()
	d := CheckTokenBudget(&tr, "sub-1", 1000, 0)
	if d.Action != BudgetActionStop || d.Completion != nil {
		t.Fatalf("%+v", d)
	}
}

func TestCheckTokenBudget_skipsZeroBudget(t *testing.T) {
	tr := NewBudgetTracker()
	d := CheckTokenBudget(&tr, "", 0, 0)
	if d.Action != BudgetActionStop || d.Completion != nil {
		t.Fatalf("%+v", d)
	}
}

func TestCheckTokenBudget_continueUnderThreshold(t *testing.T) {
	tr := NewBudgetTracker()
	d := CheckTokenBudget(&tr, "", 10_000, 100)
	if d.Action != BudgetActionContinue || d.NudgeMessage == "" {
		t.Fatalf("%+v", d)
	}
	if tr.ContinuationCount != 1 {
		t.Fatalf("tracker continuation %d", tr.ContinuationCount)
	}
}

func TestCheckTokenBudget_diminishingReturns(t *testing.T) {
	tr := BudgetTracker{
		ContinuationCount:    3,
		LastDeltaTokens:      10,
		LastGlobalTurnTokens: 5000,
		StartedAtUnixMilli:   1,
	}
	d := CheckTokenBudget(&tr, "", 100_000, 5000)
	if d.Action != BudgetActionStop || d.Completion == nil || !d.Completion.DiminishingReturns {
		t.Fatalf("%+v", d)
	}
}

func TestParseTokenBudget_shorthandStart(t *testing.T) {
	n, ok := ParseTokenBudget("+500k extra")
	if !ok || n != 500_000 {
		t.Fatalf("got %d ok=%v", n, ok)
	}
}

func TestParseTokenBudget_shorthandEnd(t *testing.T) {
	n, ok := ParseTokenBudget(`please use budget +2m.`)
	if !ok || n != 2_000_000 {
		t.Fatalf("got %d ok=%v", n, ok)
	}
}

func TestParseTokenBudget_verbose(t *testing.T) {
	n, ok := ParseTokenBudget("We should spend 3b tokens today")
	if !ok || n != 3_000_000_000 {
		t.Fatalf("got %d ok=%v", n, ok)
	}
}

func TestParseTokenBudget_none(t *testing.T) {
	if _, ok := ParseTokenBudget("no budget here"); ok {
		t.Fatal("expected false")
	}
}

func TestBudgetContinuationMessage_emDash(t *testing.T) {
	s := BudgetContinuationMessage(12, 1200, 10_000)
	if s == "" {
		t.Fatal("empty")
	}
}

func TestEstimateAttachmentRawBytesAsTokens(t *testing.T) {
	if EstimateAttachmentRawBytesAsTokens(0) != 0 {
		t.Fatal()
	}
	if EstimateAttachmentRawBytesAsTokens(8) != 2 {
		t.Fatalf("got %d", EstimateAttachmentRawBytesAsTokens(8))
	}
}

func TestEstimateResolvedSubmitTextTokens_structuredJSON(t *testing.T) {
	j := []byte(`[{"role":"user","content":"abcd"}]`)
	tok := EstimateResolvedSubmitTextTokens("structured", string(j))
	if tok <= 0 {
		t.Fatalf("structured path should return positive tokens, got %d", tok)
	}
	if got := EstimateResolvedSubmitTextTokens("bytes4", string(j)); got != EstimateUTF8BytesAsTokens(string(j)) {
		t.Fatalf("bytes4 mismatch %d vs %d", got, EstimateUTF8BytesAsTokens(string(j)))
	}
}

func TestEstimateSubmitTokenBudgetTotal(t *testing.T) {
	if n := EstimateSubmitTokenBudgetTotal("bytes4", "abcde", 4); n != 3 {
		t.Fatalf("got %d", n)
	}
}

func TestBuildSubmitTokenBudgetSnapshotPayload_envMode(t *testing.T) {
	t.Setenv(features.EnvTokenSubmitEstimateMode, "structured")
	p := BuildSubmitTokenBudgetSnapshotPayload(`[{"role":"user","content":"hi"}]`, 8, "")
	if p.Kind != SubmitTokenBudgetSnapshotKind || p.ModeDetail != "structured" {
		t.Fatalf("%+v", p)
	}
	if p.InjectRawBytes != 8 || p.TotalTokens <= 0 {
		t.Fatalf("%+v", p)
	}
}

func TestBuildSubmitTokenBudgetSnapshotPayload_modeOverride(t *testing.T) {
	t.Setenv(features.EnvTokenSubmitEstimateMode, "bytes4")
	p := BuildSubmitTokenBudgetSnapshotPayload("abcd", 0, "structured")
	if p.ModeDetail != "structured" {
		t.Fatalf("%+v", p)
	}
}

func TestEstimateMessageTokensFromTranscriptJSON_textAndToolUse(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":[{"type":"text","text":"abcd"}]},
		{"role":"assistant","content":[{"type":"tool_use","id":"x","name":"Read","input":{"p":1}}]}
	]`)
	n, err := EstimateMessageTokensFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected positive tokens, got %d", n)
	}
	base := EstimateUTF8BytesAsTokens("abcd") + EstimateUTF8BytesAsTokens(`Read{"p":1}`)
	want := (base*4 + 2) / 3
	if n != want {
		t.Fatalf("got %d want %d", n, want)
	}
}

func TestEstimateMessageTokensFromTranscriptJSON_invalid(t *testing.T) {
	_, err := EstimateMessageTokensFromTranscriptJSON([]byte(`not-json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEstimateMessageTokensFromTranscriptJSON_imageBase64Heuristic(t *testing.T) {
	longB64 := make([]byte, 4000)
	for i := range longB64 {
		longB64[i] = 'A'
	}
	raw := []byte(`[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"` + string(longB64) + `"}}]}]`)
	n, err := EstimateMessageTokensFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n <= 2000 {
		t.Fatalf("expected large base64 to raise estimate, got %d", n)
	}
}

func TestEstimateUTF8BytesAsTokens(t *testing.T) {
	if EstimateUTF8BytesAsTokens("") != 0 {
		t.Fatal()
	}
	if EstimateUTF8BytesAsTokens("abcd") != 1 {
		t.Fatalf("got %d", EstimateUTF8BytesAsTokens("abcd"))
	}
	if EstimateUTF8BytesAsTokens("abcde") != 2 {
		t.Fatalf("got %d", EstimateUTF8BytesAsTokens("abcde"))
	}
}

func TestEstimateTranscriptJSONTokens(t *testing.T) {
	if EstimateTranscriptJSONTokens([]byte(`{"x":1}`)) != 2 {
		t.Fatalf("got %d", EstimateTranscriptJSONTokens([]byte(`{"x":1}`)))
	}
}
