package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ReadWebSearchAssistantBlocks consumes SSE until EOF (after message_stop) and returns assistant
// content blocks in index order (text, server_tool_use, web_search_tool_result) for
// websearchtool.MakeOutputFromContentBlocks — analogue of accumulating event.message.content in WebSearchTool.call.
func ReadWebSearchAssistantBlocks(ctx context.Context, body io.Reader) ([]json.RawMessage, UsageDelta, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan StreamEvent, StreamBufferCapacity)
	errCh := make(chan error, 1)
	go func() {
		errCh <- StreamEvents(ctx, body, ch)
	}()

	blocks := make(map[int]*wsBlockAcc)
	var u UsageDelta
	var haveUsage bool

	for ev := range ch {
		switch ParseEventType(ev.JSON) {
		case "content_block_start":
			idx, acc, err := wsParseContentBlockStart(ev.JSON)
			if err != nil {
				cancel()
				for range ch {
				}
				<-errCh
				return nil, u, err
			}
			blocks[idx] = acc
		case "content_block_delta":
			_ = wsApplyContentBlockDelta(ev.JSON, blocks)
		case "message_delta":
			if ud, ok := ParseUsageDelta(ev.JSON); ok {
				u = ud
				haveUsage = true
			}
		case "error":
			cancel()
			for range ch {
			}
			<-errCh
			if IsPromptCacheBreakStreamJSON(ev.JSON) {
				return nil, u, ErrPromptCacheBreakDetected
			}
			return nil, u, fmtErrorfStream(ev.JSON)
		}
	}

	streamErr := <-errCh
	out, ferr := wsFinalizeBlocks(blocks)
	if ferr != nil {
		return nil, u, ferr
	}
	if streamErr != nil && streamErr != io.EOF {
		return out, u, streamErr
	}
	if !haveUsage {
		u = UsageDelta{}
	}
	return out, u, nil
}

type wsBlockAcc struct {
	typ          string
	text         strings.Builder
	blockStart   json.RawMessage // full content_block JSON from content_block_start
	serverInput  strings.Builder
}

func wsParseContentBlockStart(jsonLine []byte) (int, *wsBlockAcc, error) {
	var wrap struct {
		Index        int             `json:"index"`
		ContentBlock json.RawMessage `json:"content_block"`
	}
	if err := json.Unmarshal(jsonLine, &wrap); err != nil {
		return 0, nil, err
	}
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(wrap.ContentBlock, &probe); err != nil {
		return 0, nil, err
	}
	acc := &wsBlockAcc{typ: probe.Type, blockStart: append(json.RawMessage(nil), wrap.ContentBlock...)}
	if probe.Type == "text" {
		var tb struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(wrap.ContentBlock, &tb)
		acc.text.WriteString(tb.Text)
	}
	return wrap.Index, acc, nil
}

func wsApplyContentBlockDelta(jsonLine []byte, blocks map[int]*wsBlockAcc) error {
	var wrap struct {
		Index int `json:"index"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
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
		if b.typ == "text" {
			b.text.WriteString(wrap.Delta.Text)
		}
	case "input_json_delta":
		if b.typ == "server_tool_use" {
			var w struct {
				Delta struct {
					PartialJSON string `json:"partial_json"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(jsonLine, &w); err != nil {
				return err
			}
			b.serverInput.WriteString(w.Delta.PartialJSON)
		}
	}
	return nil
}

func wsFinalizeBlocks(blocks map[int]*wsBlockAcc) ([]json.RawMessage, error) {
	if len(blocks) == 0 {
		return nil, nil
	}
	keys := make([]int, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var out []json.RawMessage
	for _, idx := range keys {
		b := blocks[idx]
		if b == nil {
			continue
		}
		switch b.typ {
		case "text":
			raw, err := json.Marshal(map[string]any{
				"type": "text",
				"text": b.text.String(),
			})
			if err != nil {
				return out, err
			}
			out = append(out, raw)
		case "server_tool_use":
			var m map[string]any
			if err := json.Unmarshal(b.blockStart, &m); err != nil {
				return out, err
			}
			inp := strings.TrimSpace(b.serverInput.String())
			if inp == "" {
				m["input"] = map[string]any{}
			} else {
				var parsed any
				if err := json.Unmarshal([]byte(inp), &parsed); err != nil {
					return out, err
				}
				m["input"] = parsed
			}
			raw, err := json.Marshal(m)
			if err != nil {
				return out, err
			}
			out = append(out, raw)
		case "web_search_tool_result":
			out = append(out, append(json.RawMessage(nil), b.blockStart...))
		default:
			// Upstream makeOutputFromSearchResponse only folds text / server_tool_use / web_search_tool_result.
		}
	}
	return out, nil
}

func fmtErrorfStream(b []byte) error {
	return fmt.Errorf("stream error event: %s", string(b))
}
