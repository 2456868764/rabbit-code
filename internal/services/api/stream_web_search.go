package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
)

// WebSearchReadOption configures ReadWebSearchAssistantBlocks (progress + fallback query).
type WebSearchReadOption func(*webSearchReaderCfg)

type webSearchReaderCfg struct {
	fallbackQuery string
	onProgress    func(websearchtool.WebSearchProgress)
}

// WebSearchReadFallbackQuery is the user query when resolving search_results_received (upstream toolUseQueries fallback).
func WebSearchReadFallbackQuery(q string) WebSearchReadOption {
	return func(c *webSearchReaderCfg) {
		c.fallbackQuery = strings.TrimSpace(q)
	}
}

// WebSearchReadOnProgress mirrors WebSearchTool.call onProgress (query_update, search_results_received).
func WebSearchReadOnProgress(h func(websearchtool.WebSearchProgress)) WebSearchReadOption {
	return func(c *webSearchReaderCfg) {
		c.onProgress = h
	}
}

// ReadWebSearchAssistantBlocks consumes SSE until EOF (after message_stop) and returns assistant
// content blocks in index order (text, server_tool_use, web_search_tool_result) for
// websearchtool.MakeOutputFromContentBlocks — analogue of accumulating event.message.content in WebSearchTool.call.
func ReadWebSearchAssistantBlocks(ctx context.Context, body io.Reader, opts ...WebSearchReadOption) ([]json.RawMessage, UsageDelta, error) {
	var cfg webSearchReaderCfg
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}

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
	var prog wsProgressEmitState

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
			if acc != nil && acc.typ == "web_search_tool_result" && cfg.onProgress != nil {
				wsEmitWebSearchResultsProgress(acc.blockStart, cfg.fallbackQuery, &prog, cfg.onProgress)
			}
			blocks[idx] = acc
		case "content_block_delta":
			wsApplyContentBlockDeltaWithProgress(ev.JSON, blocks, &prog, cfg.onProgress)
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

type wsProgressEmitState struct {
	counter       int
	lastQueryByID map[string]string
}

func (s *wsProgressEmitState) lastQuery(id string) string {
	if s.lastQueryByID == nil {
		return ""
	}
	return s.lastQueryByID[id]
}

func (s *wsProgressEmitState) setLastQuery(id, q string) {
	if s.lastQueryByID == nil {
		s.lastQueryByID = make(map[string]string)
	}
	s.lastQueryByID[id] = q
}

func wsEmitWebSearchResultsProgress(blockJSON []byte, fallback string, st *wsProgressEmitState, on func(websearchtool.WebSearchProgress)) {
	if on == nil {
		return
	}
	var blk struct {
		Type      string          `json:"type"`
		ToolUseID string          `json:"tool_use_id"`
		Content   json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(blockJSON, &blk); err != nil {
		return
	}
	q := st.lastQuery(blk.ToolUseID)
	if q == "" {
		q = fallback
	}
	n := 0
	raw := bytes.TrimSpace(blk.Content)
	if len(raw) > 0 && raw[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err == nil {
			n = len(arr)
		}
	}
	st.counter++
	tid := strings.TrimSpace(blk.ToolUseID)
	if tid == "" {
		tid = fmt.Sprintf("search-progress-%d", st.counter)
	}
	on(websearchtool.WebSearchProgress{
		ToolUseID: tid,
		Data: websearchtool.WebSearchProgressData{
			Type:        "search_results_received",
			ResultCount: n,
			Query:       q,
		},
	})
}

type wsBlockAcc struct {
	typ          string
	text         strings.Builder
	blockStart   json.RawMessage // full content_block JSON from content_block_start
	serverInput  strings.Builder
	serverToolID string
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
	if probe.Type == "server_tool_use" {
		var sid struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(wrap.ContentBlock, &sid)
		acc.serverToolID = strings.TrimSpace(sid.ID)
	}
	return wrap.Index, acc, nil
}

func wsApplyContentBlockDeltaWithProgress(jsonLine []byte, blocks map[int]*wsBlockAcc, st *wsProgressEmitState, onProgress func(websearchtool.WebSearchProgress)) error {
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
	before := ""
	if b, ok := blocks[wrap.Index]; ok && b != nil && b.typ == "server_tool_use" {
		before = b.serverInput.String()
	}
	if err := wsApplyContentBlockDelta(jsonLine, blocks); err != nil {
		return err
	}
	if wrap.Delta.Type != "input_json_delta" || onProgress == nil {
		return nil
	}
	b, ok := blocks[wrap.Index]
	if !ok || b == nil || b.typ != "server_tool_use" {
		return nil
	}
	after := b.serverInput.String()
	if after == before {
		return nil
	}
	q, ok := websearchtool.ExtractQueryFromPartialWebSearchInputJSON(after)
	if !ok {
		return nil
	}
	id := b.serverToolID
	if id != "" && st.lastQuery(id) == q {
		return nil
	}
	if id != "" {
		st.setLastQuery(id, q)
	}
	st.counter++
	onProgress(websearchtool.WebSearchProgress{
		ToolUseID: fmt.Sprintf("search-progress-%d", st.counter),
		Data: websearchtool.WebSearchProgressData{
			Type:  "query_update",
			Query: q,
		},
	})
	return nil
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
