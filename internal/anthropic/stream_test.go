package anthropic

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestStreamEvents_Backpressure(t *testing.T) {
	sse := strings.NewReader("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n")
	ch := make(chan StreamEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = StreamEvents(ctx, sse, ch)
	}()
	select {
	case ev := <-ch:
		if !strings.Contains(string(ev.JSON), "text_delta") {
			t.Fatal(string(ev.JSON))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestStreamEvents_CRLFLineEndings(t *testing.T) {
	raw := "data: {\"type\":\"message_stop\"}\r\n\r\n"
	ch := make(chan StreamEvent, StreamBufferCapacity)
	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- StreamEvents(ctx, strings.NewReader(raw), ch)
	}()
	ev := <-ch
	if ParseEventType(ev.JSON) != "message_stop" {
		t.Fatalf("got %s", string(ev.JSON))
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestAppendThinkingDelta(t *testing.T) {
	line := []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"abc"}}`)
	var acc strings.Builder
	if err := AppendThinkingDelta(line, &acc); err != nil {
		t.Fatal(err)
	}
	if acc.String() != "abc" {
		t.Fatalf("got %q", acc.String())
	}
	var acc2 strings.Builder
	_ = AppendThinkingDelta([]byte(`{"delta":{"type":"text_delta","text":"x"}}`), &acc2)
	if acc2.String() != "" {
		t.Fatal("text_delta should not append to thinking acc")
	}
}

func TestAppendInputJSONDelta(t *testing.T) {
	var b0, b1 strings.Builder
	by := map[int]*strings.Builder{0: &b0, 2: &b1}
	line0 := []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"x\""}}`)
	if err := AppendInputJSONDelta(line0, by); err != nil {
		t.Fatal(err)
	}
	lineSkip := []byte(`{"type":"content_block_delta","index":9,"delta":{"type":"input_json_delta","partial_json":"z"}}`)
	_ = AppendInputJSONDelta(lineSkip, by)
	line1 := []byte(`{"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"}"}}`)
	_ = AppendInputJSONDelta(line1, by)
	if b0.String() != `{"x"` {
		t.Fatalf("b0 %q", b0.String())
	}
	if b1.String() != `}` {
		t.Fatalf("b1 %q", b1.String())
	}
	_ = AppendInputJSONDelta([]byte(`{"delta":{"type":"text_delta","text":"a"}}`), by)
}

func TestReadAssistantStream_WithToolInputAccumulators(t *testing.T) {
	var tool strings.Builder
	by := map[int]*strings.Builder{0: &tool}
	raw := "" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"n\\\":\"}}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	text, _, err := ReadAssistantStream(context.Background(), strings.NewReader(raw), WithToolInputAccumulators(by))
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hi" {
		t.Fatalf("text %q", text)
	}
	if tool.String() != `{"n":` {
		t.Fatalf("tool json %q", tool.String())
	}
}

func TestAppendCompactionDelta(t *testing.T) {
	line := []byte(`{"type":"content_block_delta","index":0,"delta":{"type":"compaction_delta","content":"xyz"}}`)
	var acc strings.Builder
	if err := AppendCompactionDelta(line, &acc); err != nil {
		t.Fatal(err)
	}
	if acc.String() != "xyz" {
		t.Fatalf("got %q", acc.String())
	}
}

func TestReadAssistantStream_WithCompactionAccumulator(t *testing.T) {
	raw := "" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"compaction_delta\",\"content\":\"C1\"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	var comp strings.Builder
	text, _, err := ReadAssistantStream(context.Background(), strings.NewReader(raw), WithCompactionAccumulator(&comp))
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hi" {
		t.Fatalf("text %q", text)
	}
	if comp.String() != "C1" {
		t.Fatalf("compaction %q", comp.String())
	}
}

func TestReadAssistantStream_WithThinkingAccumulator(t *testing.T) {
	raw := "" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"T1\"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	var think strings.Builder
	text, _, err := ReadAssistantStream(context.Background(), strings.NewReader(raw), WithThinkingAccumulator(&think))
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hi" {
		t.Fatalf("text %q", text)
	}
	if think.String() != "T1" {
		t.Fatalf("thinking %q", think.String())
	}
}

func TestReadAssistantStream_WithUsage(t *testing.T) {
	raw := "" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"usage\":{\"input_tokens\":10,\"output_tokens\":2}}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	text, u, err := ReadAssistantStream(context.Background(), strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hello" {
		t.Fatalf("got %q", text)
	}
	if u.InputTokens != 10 || u.OutputTokens != 2 {
		t.Fatalf("%+v", u)
	}
}

func TestStreamCancel_ClosesBody(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan StreamEvent, StreamBufferCapacity)
	done := make(chan struct{})
	go func() {
		_ = StreamEvents(ctx, pr, ch)
		close(done)
	}()
	cancel()
	_ = pw.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream reader did not finish after cancel")
	}
}
