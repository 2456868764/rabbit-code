package anthropic

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestBuildMessagesAPIMetadata_shape(t *testing.T) {
	t.Setenv(EnvClaudeCodeExtraMetadata, `{"app":"rabbit"}`)
	t.Setenv(EnvRabbitDeviceID, "dev-1")
	t.Setenv(EnvRabbitOAuthAccountUUID, "")

	cl := &Client{SessionID: "sess-9"}
	raw, err := BuildMessagesAPIMetadata(cl)
	if err != nil {
		t.Fatal(err)
	}
	var outer struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(raw, &outer); err != nil {
		t.Fatal(err)
	}
	var inner map[string]any
	if err := json.Unmarshal([]byte(outer.UserID), &inner); err != nil {
		t.Fatalf("user_id not stringified JSON: %q", outer.UserID)
	}
	if inner["app"] != "rabbit" {
		t.Fatalf("extra merge: %+v", inner)
	}
	if inner["device_id"] != "dev-1" {
		t.Fatalf("device_id: %+v", inner)
	}
	if inner["session_id"] != "sess-9" {
		t.Fatalf("session_id: %+v", inner)
	}
}

func TestMergeStreamingBody_injectsMetadata(t *testing.T) {
	cl := NewClient(http.DefaultTransport)
	cl.SessionID = "s1"
	body := MessagesStreamBody{Model: "m", MaxTokens: 1, Messages: []byte(`[]`)}
	out := cl.mergeStreamingBody(body)
	if len(out.Metadata) == 0 {
		t.Fatal("expected merged metadata")
	}
	if !strings.Contains(string(out.Metadata), `"user_id"`) {
		t.Fatalf("%s", out.Metadata)
	}
}
