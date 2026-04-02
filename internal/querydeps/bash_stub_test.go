package querydeps

import (
	"context"
	"encoding/json"
	"testing"
)

func TestBashStubToolRunner(t *testing.T) {
	var tr BashStubToolRunner
	out, err := tr.RunTool(context.Background(), "bash", []byte(`{"cmd":"echo hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["ok"] != true || m["stub"] != "bash" {
		t.Fatalf("got %s", out)
	}
	if _, err := tr.RunTool(context.Background(), "other", nil); err == nil {
		t.Fatal("expected error")
	}
}
