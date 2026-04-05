package query

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TrimTranscriptPrefixWhileOverBudget drops leading messages (one per round) while len(msgs) > maxBytes.
func TrimTranscriptPrefixWhileOverBudget(msgs json.RawMessage, maxBytes, maxRounds int) (out json.RawMessage, rounds int, err error) {
	out = msgs
	if maxBytes <= 0 || maxRounds <= 0 {
		return out, 0, nil
	}
	for len(out) > maxBytes && maxRounds > 0 {
		next, err := SnipDropFirstMessages(out, 1)
		if err != nil {
			return msgs, rounds, err
		}
		var arr []json.RawMessage
		if err := json.Unmarshal(next, &arr); err != nil {
			return msgs, rounds, err
		}
		if len(arr) == 0 {
			break
		}
		out = next
		rounds++
		maxRounds--
	}
	return out, rounds, nil
}

// StripCacheControlFromMessagesJSON returns a copy of the messages array JSON with every
// "cache_control" key removed recursively (mirrors stripCacheControl in promptCacheBreakDetection.ts).
func StripCacheControlFromMessagesJSON(raw json.RawMessage) (out json.RawMessage, changed bool, err error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, false, errors.New("query: strip cache: empty messages")
	}
	var top interface{}
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, false, err
	}
	stripped, removed := stripCacheControlWalk(top)
	enc, err := json.Marshal(stripped)
	if err != nil {
		return nil, false, err
	}
	return json.RawMessage(enc), removed, nil
}

// RemapPromptCacheBreakpointsForSkipCacheWrite strips all cache_control markers, then adds exactly one
// ephemeral breakpoint on the message at index len(messages)-2 when len>=2 (query.ts / claude.ts addCacheBreakpoints
// with skipCacheWrite: fork/side paths avoid writing a new tail into KVCC). When len(messages)<2, leaves
// no cache_control (matches TS markerIndex < 0 case).
func RemapPromptCacheBreakpointsForSkipCacheWrite(raw json.RawMessage) (json.RawMessage, error) {
	stripped, _, err := StripCacheControlFromMessagesJSON(raw)
	if err != nil {
		return nil, err
	}
	var arr []interface{}
	if err := json.Unmarshal(stripped, &arr); err != nil {
		return nil, err
	}
	if len(arr) < 2 {
		return stripped, nil
	}
	markerIndex := len(arr) - 2
	msg, ok := arr[markerIndex].(map[string]interface{})
	if !ok {
		return stripped, nil
	}
	addEphemeralCacheControlToMessageContent(msg)
	out, err := json.Marshal(arr)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func addEphemeralCacheControlToMessageContent(m map[string]interface{}) {
	content, ok := m["content"]
	if !ok {
		return
	}
	ephemeral := map[string]interface{}{"type": "ephemeral"}
	switch c := content.(type) {
	case string:
		m["content"] = []interface{}{
			map[string]interface{}{
				"type":           "text",
				"text":           c,
				"cache_control": ephemeral,
			},
		}
	case []interface{}:
		if len(c) == 0 {
			return
		}
		last, ok := c[len(c)-1].(map[string]interface{})
		if !ok {
			return
		}
		last["cache_control"] = ephemeral
	default:
		return
	}
}

func stripCacheControlWalk(v interface{}) (interface{}, bool) {
	switch x := v.(type) {
	case map[string]interface{}:
		removed := false
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			if k == "cache_control" {
				removed = true
				continue
			}
			nv, r := stripCacheControlWalk(val)
			if r {
				removed = true
			}
			out[k] = nv
		}
		return out, removed
	case []interface{}:
		out := make([]interface{}, len(x))
		removed := false
		for i := range x {
			nv, r := stripCacheControlWalk(x[i])
			if r {
				removed = true
			}
			out[i] = nv
		}
		return out, removed
	default:
		return v, false
	}
}

// UserTextHintFlags gates optional user-message hints (P5.F.3–F.5, headless).
type UserTextHintFlags struct {
	ContextCollapse bool
	Ultrathink      bool
	Ultraplan       bool
	SessionRestore  bool
}

const (
	hintContextCollapseSuffix = "\n\n[CONTEXT_COLLAPSE: prefer collapsing stale context; avoid repeating large verbatim dumps.]\n"
	hintUltrathinkPrefix      = "[ULTRATHINK: reason step-by-step before answering.]\n\n"
	hintUltraplanSuffix       = "\n\n[ULTRAPLAN: outline a short plan before executing tool calls.]"
	hintSessionRestoreSuffix  = "\n\n[SESSION_RESTORE: prefer restoring durable session context over re-deriving from scratch.]"
)

// ApplyUserTextHints mutates the resolved user payload before InitialUserMessagesJSON (engine Submit path).
func ApplyUserTextHints(text string, f UserTextHintFlags) string {
	if text == "" {
		return text
	}
	out := text
	if f.Ultrathink {
		out = hintUltrathinkPrefix + out
	}
	if f.ContextCollapse {
		out = out + hintContextCollapseSuffix
	}
	if f.Ultraplan {
		out = out + hintUltraplanSuffix
	}
	if f.SessionRestore {
		out = out + hintSessionRestoreSuffix
	}
	return out
}

// FormatHeadlessModeTags lists active headless input modes for TUI/telemetry (comma-separated, stable order).
func FormatHeadlessModeTags(f UserTextHintFlags) string {
	var parts []string
	if f.ContextCollapse {
		parts = append(parts, "context_collapse")
	}
	if f.Ultrathink {
		parts = append(parts, "ultrathink")
	}
	if f.Ultraplan {
		parts = append(parts, "ultraplan")
	}
	if f.SessionRestore {
		parts = append(parts, "session_restore")
	}
	return strings.Join(parts, ",")
}

// LoadTemplateMarkdownAppendix reads <dir>/<name>.md for each name and returns a single appendix string (P5.F.7 body load).
func LoadTemplateMarkdownAppendix(dir string, names []string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", nil
	}
	var b strings.Builder
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		if strings.Contains(name, string(os.PathSeparator)) || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			return "", fmt.Errorf("query: invalid template name %q", rawName)
		}
		p := filepath.Join(dir, name+".md")
		raw, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "\n\n## Template %s\n%s", name, string(raw))
	}
	return b.String(), nil
}
