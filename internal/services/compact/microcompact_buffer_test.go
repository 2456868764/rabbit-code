package compact

import (
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

func TestMicrocompactEditBuffer_implementsQuerydepsMarker(t *testing.T) {
	var m querydeps.MicrocompactAPIStateMarker = &MicrocompactEditBuffer{}
	m.MarkToolsSentToAPIState()
}

func TestMicrocompactEditBuffer_flow(t *testing.T) {
	var b MicrocompactEditBuffer
	b.SetPendingCacheEdits(json.RawMessage(`{"x":1}`))
	got := b.ConsumePendingCacheEdits()
	if string(got) != `{"x":1}` {
		t.Fatalf("pending %s", got)
	}
	if b.ConsumePendingCacheEdits() != nil {
		t.Fatal("consume clears")
	}
	b.PinCacheEdits(2, json.RawMessage(`{"pin":true}`))
	p := b.GetPinnedCacheEdits()
	if len(p) != 1 || p[0].UserMessageIndex != 2 {
		t.Fatalf("%+v", p)
	}
	b.MarkToolsSentToAPIState()
	if !b.ToolsSentToAPI() {
		t.Fatal("tools sent")
	}
	b.ResetMicrocompactState()
	if len(b.GetPinnedCacheEdits()) != 0 {
		t.Fatal("reset pinned")
	}
	if b.ToolsSentToAPI() {
		t.Fatal("reset flag")
	}
}
