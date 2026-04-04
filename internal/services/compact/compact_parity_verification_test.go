package compact

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Execution list §4 / §5 / §13 / §15: constants, strip reinject, prompt smoke, grouping JSON vs typed.

func TestCompactConstantsMatchTS_compactTs(t *testing.T) {
	// compact.ts POST_COMPACT_* and related caps @ a8a678c
	if PostCompactMaxFilesToRestore != 5 ||
		PostCompactTokenBudget != 50_000 ||
		PostCompactMaxTokensPerFile != 5_000 ||
		PostCompactMaxTokensPerSkill != 5_000 ||
		PostCompactSkillsTokenBudget != 25_000 ||
		MaxCompactStreamingRetries != 2 ||
		MaxPTLRetries != 3 {
		t.Fatal("POST_COMPACT_* / retry caps drift from compact.ts")
	}
	if ErrorMessageNotEnoughMessages != "Not enough messages to compact." ||
		ErrorMessagePromptTooLong != "Conversation too long. Press esc twice to go up a few messages and try again." ||
		ErrorMessageUserAbort != "API Error: Request was aborted." ||
		ErrorMessageIncompleteResponse != "Compaction interrupted · This may be due to network issues — please try again." {
		t.Fatal("ERROR_MESSAGE_* drift from compact.ts")
	}
	if PTLRetryMarker != "[earlier conversation truncated for compaction retry]" {
		t.Fatalf("PTLRetryMarker: %q", PTLRetryMarker)
	}
	if CompactToolUseDenyMessage != "Tool use is not allowed during compaction" {
		t.Fatalf("createCompactCanUseTool message drift: %q", CompactToolUseDenyMessage)
	}
}

func TestStripReinjectedAttachmentsFromTranscriptJSON_featureGated(t *testing.T) {
	t.Setenv(features.EnvExperimentalSkillSearch, "")
	raw := []byte(`[
	  {"type":"user","message":{"content":[{"type":"text","text":"hi"}]}},
	  {"type":"attachment","attachment":{"type":"skill_discovery"}},
	  {"type":"attachment","attachment":{"type":"skill_listing"}}
	]`)
	out, err := StripReinjectedAttachmentsFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(raw) {
		t.Fatalf("expected no-op when feature off, got %s", out)
	}

	t.Setenv(features.EnvExperimentalSkillSearch, "1")
	out2, err := StripReinjectedAttachmentsFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	var arr []interface{}
	if err := json.Unmarshal(out2, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 {
		t.Fatalf("expected only user message, got %d elems", len(arr))
	}
}

func TestPromptCompact_smokeMatchesTSExports(t *testing.T) {
	if !strings.Contains(GetCompactPrompt(""), "<analysis>") {
		t.Fatal("GetCompactPrompt should include analysis tags (prompt.ts)")
	}
	if !strings.Contains(GetPartialCompactPrompt("", PartialCompactFrom), "<analysis>") {
		t.Fatal("GetPartialCompactPrompt(from) should include analysis tags")
	}
	s := FormatCompactSummary("<summary>hello</summary>")
	if !strings.Contains(s, "hello") || !strings.Contains(s, "Summary:") {
		t.Fatalf("FormatCompactSummary: %q", s)
	}
	u := GetCompactUserSummaryMessage("sum", false, "/tmp/t.json", false)
	if !strings.Contains(u, "sum") {
		t.Fatalf("GetCompactUserSummaryMessage: %q", u)
	}
}

func TestGroupRawMessagesByAPIRound_matchesTypedGrouping(t *testing.T) {
	ms := []ApiRoundMessage{
		{Type: "user"},
		assistantMsg("r1"),
		assistantMsg("r1"),
		assistantMsg("r2"),
	}
	typed := GroupMessagesByApiRound(ms)

	var lines []json.RawMessage
	for _, m := range ms {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, b)
	}
	rawGroups := GroupRawMessagesByAPIRound(lines)

	if len(rawGroups) != len(typed) {
		t.Fatalf("group count typed=%d raw=%d", len(typed), len(rawGroups))
	}
	for i := range typed {
		if len(rawGroups[i]) != len(typed[i]) {
			t.Fatalf("group %d len typed=%d raw=%d", i, len(typed[i]), len(rawGroups[i]))
		}
	}
}

func TestTimeBasedMCClearedMessage_matchesMicroCompactTs(t *testing.T) {
	if TimeBasedMCClearedMessage != "[Old tool result content cleared]" {
		t.Fatalf("TIME_BASED_MC_CLEARED_MESSAGE drift: %q", TimeBasedMCClearedMessage)
	}
}

func TestAutoCompactBufferConstants_matchAutoCompactTs(t *testing.T) {
	if AutocompactBufferTokens != 13_000 ||
		WarningThresholdBufferTokens != 20_000 ||
		ErrorThresholdBufferTokens != 20_000 ||
		ManualCompactBufferTokens != 3_000 {
		t.Fatal("autoCompact.ts buffer token constants drift")
	}
}
