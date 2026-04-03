package memdir

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/2456868764/rabbit-code/internal/query"
)

// CountModelVisibleMessagesSince counts user + assistant messages after the message with sinceUUID (extractMemories.ts).
// If sinceUUID is empty, counts all model-visible messages. If sinceUUID is not found, falls back to full count.
func CountModelVisibleMessagesSince(msgs json.RawMessage, sinceUUID, uuidField string) int {
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}
	arr, err := parseTopMessagesArray(msgs)
	if err != nil || len(arr) == 0 {
		return 0
	}
	start := 0
	if sinceUUID != "" {
		found := false
		for i, raw := range arr {
			id := topLevelStringField(raw, uuidField)
			if id == sinceUUID {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			start = 0
		}
	}
	n := 0
	for i := start; i < len(arr); i++ {
		role := topLevelStringField(arr[i], "role")
		if role == "user" || role == "assistant" {
			n++
		}
	}
	return n
}

// HasMemoryWritesSince is true if any assistant message after sinceUUID contains Write/Edit tool_use targeting autoMemDir.
func HasMemoryWritesSince(msgs json.RawMessage, sinceUUID, autoMemDir, uuidField string) bool {
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}
	arr, err := parseTopMessagesArray(msgs)
	if err != nil {
		return false
	}
	foundStart := sinceUUID == ""
	for _, raw := range arr {
		if !foundStart {
			if topLevelStringField(raw, uuidField) == sinceUUID {
				foundStart = true
			}
			continue
		}
		if topLevelStringField(raw, "role") != "assistant" {
			continue
		}
		for _, fp := range toolUseFilePathsFromAssistantMessage(raw, toolNameWrite, toolNameEdit) {
			if IsAutoMemPath(fp, autoMemDir) {
				return true
			}
		}
	}
	return false
}

// WrittenMemoryPathsFromTranscriptSuffix returns Write/Edit file_path values from assistant messages at indices >= startIdx.
func WrittenMemoryPathsFromTranscriptSuffix(msgs json.RawMessage, startIdx int) []string {
	arr, err := parseTopMessagesArray(msgs)
	if err != nil {
		return nil
	}
	if startIdx < 0 {
		startIdx = 0
	}
	var out []string
	seen := make(map[string]struct{})
	for i := startIdx; i < len(arr); i++ {
		if topLevelStringField(arr[i], "role") != "assistant" {
			continue
		}
		for _, fp := range toolUseFilePathsFromAssistantMessage(arr[i], toolNameWrite, toolNameEdit) {
			fp = strings.TrimSpace(fp)
			if fp == "" {
				continue
			}
			if _, ok := seen[fp]; ok {
				continue
			}
			seen[fp] = struct{}{}
			out = append(out, fp)
		}
	}
	return out
}

// LastEmbeddedMessageUUID returns the uuidField value on the last top-level message, if any.
func LastEmbeddedMessageUUID(msgs json.RawMessage, uuidField string) string {
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}
	arr, err := parseTopMessagesArray(msgs)
	if err != nil || len(arr) == 0 {
		return ""
	}
	return topLevelStringField(arr[len(arr)-1], uuidField)
}

// TranscriptMessageCount returns the number of top-level messages in the API array.
func TranscriptMessageCount(msgs json.RawMessage) int {
	arr, err := parseTopMessagesArray(msgs)
	if err != nil {
		return 0
	}
	return len(arr)
}

const (
	toolNameWrite = "Write"
	toolNameEdit  = "Edit"
)

func parseTopMessagesArray(msgs json.RawMessage) ([]json.RawMessage, error) {
	raw := bytes.TrimSpace(msgs)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func topLevelStringField(msg json.RawMessage, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(msg, &m); err != nil {
		return ""
	}
	v, ok := m[field]
	if !ok {
		return ""
	}
	var s string
	_ = json.Unmarshal(v, &s)
	return s
}

func toolUseFilePathsFromAssistantMessage(msg json.RawMessage, toolNames ...string) []string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(msg, &m); err != nil {
		return nil
	}
	rawContent, ok := m["content"]
	if !ok {
		return nil
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(rawContent, &blocks); err != nil {
		return nil
	}
	var paths []string
	for _, b := range blocks {
		t, _ := jsonStringFieldFromMap(b, "type")
		if t != "tool_use" {
			continue
		}
		name, _ := jsonStringFieldFromMap(b, "name")
		if !toolNameMatches(name, toolNames) {
			continue
		}
		inRaw, ok := b["input"]
		if !ok {
			continue
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(inRaw, &input); err != nil {
			continue
		}
		if fpRaw, ok := input["file_path"]; ok {
			var fp string
			_ = json.Unmarshal(fpRaw, &fp)
			if fp != "" {
				paths = append(paths, fp)
			}
		}
	}
	return paths
}

func toolNameMatches(name string, allowed []string) bool {
	for _, n := range allowed {
		if strings.EqualFold(strings.TrimSpace(name), n) {
			return true
		}
	}
	return false
}

func jsonStringFieldFromMap(m map[string]json.RawMessage, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", false
	}
	return s, true
}
