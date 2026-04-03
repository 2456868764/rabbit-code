package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// StreamToolUse is one tool_use block assembled from the Messages stream.
type StreamToolUse struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// AssistantStreamTurn is text + tool uses + stop metadata from one streamed assistant message.
type AssistantStreamTurn struct {
	Text       string
	ToolUses   []StreamToolUse
	StopReason string
}

type streamContentBlock struct {
	kind      string // "text" | "tool_use"
	text      *strings.Builder
	id, name  string
	toolInput *strings.Builder
}

// ReadAssistantStreamTurn consumes SSE until message_stop and assembles text blocks (in index order)
// and tool_use blocks (id, name, streamed input JSON). Options match ReadAssistantStream.
func ReadAssistantStreamTurn(ctx context.Context, body io.Reader, opts ...ReadAssistantOption) (AssistantStreamTurn, UsageDelta, error) {
	var cfg readAssistantConfig
	for _, o := range opts {
		o(&cfg)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan StreamEvent, StreamBufferCapacity)
	errCh := make(chan error, 1)
	go func() {
		errCh <- StreamEvents(ctx, body, ch)
	}()

	blocks := make(map[int]*streamContentBlock)
	var u UsageDelta
	var haveUsage bool
	var stopReason string

	for ev := range ch {
		switch ParseEventType(ev.JSON) {
		case "content_block_start":
			idx, blk, err := parseContentBlockStart(ev.JSON)
			if err != nil {
				cancel()
				for range ch {
				}
				<-errCh
				return AssistantStreamTurn{}, u, err
			}
			blocks[idx] = blk
		case "content_block_delta":
			if err := applyContentBlockDelta(ev.JSON, blocks, &cfg); err != nil {
				cancel()
				for range ch {
				}
				<-errCh
				return AssistantStreamTurn{}, u, err
			}
		case "message_delta":
			if ud, ok := ParseUsageDelta(ev.JSON); ok {
				u = ud
				haveUsage = true
			}
			if sr, ok := parseStopReason(ev.JSON); ok {
				stopReason = sr
			}
		case "error":
			var evErr error
			if IsPromptCacheBreakStreamJSON(ev.JSON) {
				if cfg.onPromptCacheBreak != nil {
					cfg.onPromptCacheBreak()
				}
				evErr = ErrPromptCacheBreakDetected
			} else {
				evErr = fmt.Errorf("stream error event: %s", string(ev.JSON))
			}
			cancel()
			for range ch {
			}
			<-errCh
			turn := assembleTurn(blocks, stopReason)
			return turn, u, evErr
		}
	}

	streamErr := <-errCh
	turn := assembleTurn(blocks, stopReason)
	if streamErr != nil && streamErr != io.EOF {
		return turn, u, streamErr
	}
	if !haveUsage {
		u = UsageDelta{}
	}
	return turn, u, nil
}

func parseContentBlockStart(jsonLine []byte) (int, *streamContentBlock, error) {
	var wrap struct {
		Index        int `json:"index"`
		ContentBlock struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content_block"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return 0, nil, err
	}
	switch wrap.ContentBlock.Type {
	case "text":
		return wrap.Index, &streamContentBlock{kind: "text", text: new(strings.Builder)}, nil
	case "tool_use":
		tb := new(strings.Builder)
		raw := bytesTrimSpaceJSON(wrap.ContentBlock.Input)
		// Empty object is a placeholder; streamed input_json_delta supplies the real JSON.
		if len(raw) > 0 && string(raw) != "null" && string(raw) != "{}" {
			tb.Write(raw)
		}
		return wrap.Index, &streamContentBlock{
			kind:      "tool_use",
			id:        wrap.ContentBlock.ID,
			name:      wrap.ContentBlock.Name,
			toolInput: tb,
		}, nil
	default:
		// thinking / compaction / unknown: ignore for turn assembly (deltas may still arrive)
		return wrap.Index, &streamContentBlock{kind: wrap.ContentBlock.Type, text: new(strings.Builder)}, nil
	}
}

func bytesTrimSpaceJSON(b json.RawMessage) []byte {
	s := strings.TrimSpace(string(b))
	return []byte(s)
}

func applyContentBlockDelta(jsonLine []byte, blocks map[int]*streamContentBlock, cfg *readAssistantConfig) error {
	var wrap struct {
		Index int `json:"index"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return err
	}
	b, ok := blocks[wrap.Index]
	if !ok || b == nil {
		return nil
	}
	switch wrap.Delta.Type {
	case "text_delta":
		if b.kind == "text" && b.text != nil {
			b.text.WriteString(wrap.Delta.Text)
		}
	case "input_json_delta":
		if b.kind == "tool_use" && b.toolInput != nil {
			var w struct {
				Delta struct {
					PartialJSON string `json:"partial_json"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(jsonLine, &w); err != nil {
				return err
			}
			b.toolInput.WriteString(w.Delta.PartialJSON)
		}
	case "thinking_delta":
		if cfg.thinking != nil {
			_ = AppendThinkingDelta(jsonLine, cfg.thinking)
		}
	case "compaction_delta":
		if cfg.compaction != nil {
			_ = AppendCompactionDelta(jsonLine, cfg.compaction)
		}
	}
	if len(cfg.toolInputByBlock) > 0 {
		_ = AppendInputJSONDelta(jsonLine, cfg.toolInputByBlock)
	}
	return nil
}

func parseStopReason(jsonLine []byte) (string, bool) {
	var wrap struct {
		Delta struct {
			StopReason string `json:"stop_reason"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return "", false
	}
	if wrap.Delta.StopReason == "" {
		return "", false
	}
	return wrap.Delta.StopReason, true
}

func assembleTurn(blocks map[int]*streamContentBlock, stopReason string) AssistantStreamTurn {
	if len(blocks) == 0 {
		return AssistantStreamTurn{StopReason: stopReason}
	}
	keys := make([]int, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var textParts []string
	var tools []StreamToolUse
	for _, idx := range keys {
		b := blocks[idx]
		if b == nil {
			continue
		}
		switch b.kind {
		case "text":
			if b.text != nil {
				if s := b.text.String(); s != "" {
					textParts = append(textParts, s)
				}
			}
		case "tool_use":
			raw := []byte("{}")
			if b.toolInput != nil {
				s := strings.TrimSpace(b.toolInput.String())
				if s != "" {
					raw = []byte(s)
				}
			}
			tools = append(tools, StreamToolUse{ID: b.id, Name: b.name, Input: json.RawMessage(raw)})
		}
	}
	return AssistantStreamTurn{
		Text:       strings.Join(textParts, ""),
		ToolUses:   tools,
		StopReason: stopReason,
	}
}
