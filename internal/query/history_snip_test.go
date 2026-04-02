package query

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTrimTranscriptPrefixWhileOverBudget(t *testing.T) {
	raw, err := InitialUserMessagesJSON("u")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = AppendAssistantTextMessage(raw, strings.Repeat("a", 500))
	if err != nil {
		t.Fatal(err)
	}
	out, rounds, err := TrimTranscriptPrefixWhileOverBudget(raw, 200, 3)
	if err != nil {
		t.Fatal(err)
	}
	if rounds < 1 {
		t.Fatalf("rounds=%d", rounds)
	}
	if len(out) > len(raw) {
		t.Fatal("expected shrink")
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(out, &arr); err != nil || len(arr) == 0 {
		t.Fatalf("bad out %s", out)
	}
}
