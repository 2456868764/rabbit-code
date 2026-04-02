package query

import (
	"encoding/json"
	"testing"
)

func TestSnipDropFirstMessages(t *testing.T) {
	raw, err := InitialUserMessagesJSON("a")
	if err != nil {
		t.Fatal(err)
	}
	raw, err = AppendAssistantTextMessage(raw, "b")
	if err != nil {
		t.Fatal(err)
	}
	out, err := SnipDropFirstMessages(raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 {
		t.Fatalf("len=%d", len(arr))
	}
	if arr[0]["role"] != "assistant" {
		t.Fatalf("%v", arr[0])
	}
	out2, err := SnipDropFirstMessages(raw, 99)
	if err != nil {
		t.Fatal(err)
	}
	var a2 []any
	if err := json.Unmarshal(out2, &a2); err != nil || len(a2) != 0 {
		t.Fatalf("want empty array got %s err=%v", out2, err)
	}
	if _, err := SnipDropFirstMessages(raw, -1); err != ErrSnipInvalidN {
		t.Fatalf("got %v", err)
	}
}
