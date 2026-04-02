package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
)

// StreamBufferCapacity bounds queued SSE JSON payloads per stream (AC4-1c / P4.1.2).
const StreamBufferCapacity = 64

// StreamEvent is one SSE `data:` JSON line from the Messages stream.
type StreamEvent struct {
	JSON []byte
}

// AssistantTextDelta accumulates text_delta parts.
type AssistantTextDelta struct {
	Text string
}

// UsageDelta carries token usage from message_delta when present.
type UsageDelta struct {
	InputTokens              int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
	OutputTokens             int64
}

// StreamEvents reads Anthropic-style SSE (data: lines) from r until EOF or ctx cancel.
// Events are sent on ch; sender blocks when ch is full (backpressure).
func StreamEvents(ctx context.Context, r io.Reader, ch chan<- StreamEvent) error {
	defer close(ch)
	sc := bufio.NewScanner(r)
	// Large lines for JSON payloads.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return err
			}
			return nil
		}
		line := bytes.TrimSuffix(sc.Bytes(), []byte("\r"))
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte(":")) {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(line[5:])
		if len(payload) == 0 || string(payload) == "[DONE]" {
			continue
		}
		ev := StreamEvent{JSON: append([]byte(nil), payload...)}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- ev:
		}
	}
}

// ParseEventType extracts top-level "type" field without full decode.
func ParseEventType(jsonLine []byte) string {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(jsonLine, &probe); err != nil {
		return ""
	}
	return probe.Type
}

// AppendTextDelta updates acc from a content_block_delta event JSON.
func AppendTextDelta(jsonLine []byte, acc *strings.Builder) error {
	var wrap struct {
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return err
	}
	if wrap.Delta.Type == "text_delta" {
		acc.WriteString(wrap.Delta.Text)
	}
	return nil
}

// AppendThinkingDelta appends thinking stream fragments from a content_block_delta event (interleaved thinking beta).
func AppendThinkingDelta(jsonLine []byte, acc *strings.Builder) error {
	if acc == nil {
		return nil
	}
	var wrap struct {
		Delta struct {
			Type     string `json:"type"`
			Thinking string `json:"thinking"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return err
	}
	if wrap.Delta.Type == "thinking_delta" {
		acc.WriteString(wrap.Delta.Thinking)
	}
	return nil
}

// AppendCompactionDelta appends compaction block fragments from a content_block_delta event (context-management beta).
func AppendCompactionDelta(jsonLine []byte, acc *strings.Builder) error {
	if acc == nil {
		return nil
	}
	var wrap struct {
		Delta struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return err
	}
	if wrap.Delta.Type == "compaction_delta" {
		acc.WriteString(wrap.Delta.Content)
	}
	return nil
}

// AppendInputJSONDelta appends partial_json from input_json_delta events into byIndex[event.index] when that entry exists and is non-nil.
func AppendInputJSONDelta(jsonLine []byte, byIndex map[int]*strings.Builder) error {
	if len(byIndex) == 0 {
		return nil
	}
	var wrap struct {
		Index int `json:"index"`
		Delta struct {
			Type        string `json:"type"`
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return err
	}
	if wrap.Delta.Type != "input_json_delta" {
		return nil
	}
	if acc, ok := byIndex[wrap.Index]; ok && acc != nil {
		acc.WriteString(wrap.Delta.PartialJSON)
	}
	return nil
}

// ParseUsageDelta extracts usage from message_delta JSON.
func ParseUsageDelta(jsonLine []byte) (UsageDelta, bool) {
	var wrap struct {
		Delta struct {
			Usage struct {
				InputTokens              int64 `json:"input_tokens"`
				CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
				CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
				OutputTokens             int64 `json:"output_tokens"`
			} `json:"usage"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return UsageDelta{}, false
	}
	u := wrap.Delta.Usage
	if u.InputTokens == 0 && u.OutputTokens == 0 && u.CacheReadInputTokens == 0 && u.CacheCreationInputTokens == 0 {
		return UsageDelta{}, false
	}
	return UsageDelta{
		InputTokens:              u.InputTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:     u.CacheReadInputTokens,
		OutputTokens:             u.OutputTokens,
	}, true
}
