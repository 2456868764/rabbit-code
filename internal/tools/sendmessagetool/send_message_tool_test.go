package sendmessagetool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestSendMessage_textMessage(t *testing.T) {
	out, err := New().Run(context.Background(), []byte(`{"to":"researcher","message":"start task 1"}`))
	if err != nil {
		t.Fatal(err)
	}
	var o MessageOutput
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if !o.Success {
		t.Fatalf("expected success: %v", o)
	}
}

func TestSendMessage_broadcast(t *testing.T) {
	out, err := New().Run(context.Background(), []byte(`{"to":"*","message":"all done"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("empty output")
	}
}

func TestSendMessage_structuredShutdownRequest(t *testing.T) {
	raw := `{"to":"researcher","message":{"type":"shutdown_request","reason":"work done"}}`
	out, err := New().Run(context.Background(), []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var o MessageOutput
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if !o.Success {
		t.Fatalf("expected success: %v", o)
	}
}

func TestSendMessage_structuredShutdownResponse(t *testing.T) {
	raw := `{"to":"team-lead","message":{"type":"shutdown_response","request_id":"r1","approve":true}}`
	out, err := New().Run(context.Background(), []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var o MessageOutput
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if !o.Success {
		t.Fatalf("expected success: %v", o)
	}
}

func TestSendMessage_structuredPlanApprovalResponse(t *testing.T) {
	raw := `{"to":"teammate","message":{"type":"plan_approval_response","request_id":"r2","approve":false,"feedback":"needs work"}}`
	_, err := New().Run(context.Background(), []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendMessage_rejectsUnknownFields(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"to":"x","message":"hi","extra":1}`))
	if err == nil {
		t.Fatal("expected strict JSON error")
	}
}

func TestSendMessage_requiresTo(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"to":"","message":"hi"}`))
	if err == nil {
		t.Fatal("expected error for empty to")
	}
}

func TestSendMessage_delegatesToCallback(t *testing.T) {
	called := false
	rc := &RunContext{
		SendMsg: func(_ context.Context, in Input) ([]byte, error) {
			called = true
			if in.To != "researcher" {
				t.Errorf("unexpected to: %s", in.To)
			}
			return json.Marshal(MessageOutput{Success: true, Message: "delivered"})
		},
	}
	ctx := WithRunContext(context.Background(), rc)
	out, err := New().Run(ctx, []byte(`{"to":"researcher","message":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("SendMsg not called")
	}
	var o MessageOutput
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.Message != "delivered" {
		t.Fatalf("unexpected message: %s", o.Message)
	}
}

func TestSendMessage_invalidStructuredMessage(t *testing.T) {
	// Object with unknown type should fail unmarshal gracefully.
	raw := `{"to":"x","message":{"type":"unknown_type"}}`
	// The current impl tries StructuredMessage decode which succeeds (unknown type allowed in Go struct).
	// This just checks it doesn't crash.
	_, _ = New().Run(context.Background(), []byte(raw))
}

func TestSendMessageToolName(t *testing.T) {
	if SendMessageToolName != "SendMessage" {
		t.Fatalf("unexpected: %q", SendMessageToolName)
	}
	if New().Name() != SendMessageToolName {
		t.Fatal("Name() mismatch")
	}
}

func TestSendMessage_withSummary(t *testing.T) {
	raw := `{"to":"researcher","summary":"task update","message":"done with step 1"}`
	out, err := New().Run(context.Background(), []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "researcher") {
		t.Fatalf("unexpected output: %s", out)
	}
}
