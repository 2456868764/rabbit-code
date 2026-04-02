package query

import (
	"encoding/json"
	"testing"
)

func TestAppendUserThenAssistant(t *testing.T) {
	msgs, err := InitialUserMessagesJSON("hi")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err = AppendAssistantTextMessage(msgs, "hello")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err = AppendUserTextMessage(msgs, "again")
	if err != nil {
		t.Fatal(err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(msgs, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 3 {
		t.Fatalf("len %d", len(arr))
	}
}
