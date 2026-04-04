package compact

import (
	"encoding/json"
	"time"
)

const imageMaxTokenSizeMicrocompact = 2000

// MaybeTimeBasedMicrocompactJSON mirrors maybeTimeBasedMicrocompact message mutation (CC-shaped transcript:
// array of {type,timestamp?,message:{content:[...]}}). Returns original bytes unchanged when the trigger does not fire,
// clearSet is empty, or no token savings. Side-effect free; use RunMaybeTimeBasedMicrocompactJSON for suppress/reset.
func MaybeTimeBasedMicrocompactJSON(messagesJSON []byte, querySource string, now time.Time) (out []byte, tokensSaved int, changed bool, err error) {
	var arr []interface{}
	if err := json.Unmarshal(messagesJSON, &arr); err != nil {
		return nil, 0, false, err
	}
	triggerMsgs := ccMessagesTriggerView(arr)
	ev := EvaluateTimeBasedTrigger(triggerMsgs, querySource, now)
	if ev == nil {
		return append([]byte(nil), messagesJSON...), 0, false, nil
	}
	ids := collectCompactableToolUseIDsFromCCArray(arr)
	if len(ids) == 0 {
		return append([]byte(nil), messagesJSON...), 0, false, nil
	}
	keepN := ev.Config.KeepRecent
	if keepN < 1 {
		keepN = 1
	}
	keep := make(map[string]struct{})
	start := len(ids) - keepN
	if start < 0 {
		start = 0
	}
	for i := start; i < len(ids); i++ {
		keep[ids[i]] = struct{}{}
	}
	clearSet := make(map[string]struct{})
	for _, id := range ids {
		if _, ok := keep[id]; !ok {
			clearSet[id] = struct{}{}
		}
	}
	if len(clearSet) == 0 {
		return append([]byte(nil), messagesJSON...), 0, false, nil
	}
	tokensSaved = mutateUserToolResultsInCCArray(arr, clearSet)
	if tokensSaved == 0 {
		return append([]byte(nil), messagesJSON...), 0, false, nil
	}
	out, err = json.Marshal(arr)
	if err != nil {
		return nil, 0, false, err
	}
	return out, tokensSaved, true, nil
}

// RunMaybeTimeBasedMicrocompactJSON runs MaybeTimeBasedMicrocompactJSON and on success mirrors TS side effects:
// SuppressCompactWarning + ResetMicrocompactStateIfAny(buf). Caller should invoke prompt-cache deletion when applicable.
func RunMaybeTimeBasedMicrocompactJSON(messagesJSON []byte, querySource string, now time.Time, buf *MicrocompactEditBuffer) (out []byte, tokensSaved int, changed bool, err error) {
	out, tokensSaved, changed, err = MaybeTimeBasedMicrocompactJSON(messagesJSON, querySource, now)
	if err != nil || !changed || tokensSaved <= 0 {
		return out, tokensSaved, changed, err
	}
	SuppressCompactWarning()
	ResetMicrocompactStateIfAny(buf)
	return out, tokensSaved, changed, nil
}

func ccMessagesTriggerView(arr []interface{}) []TimeBasedCCMessage {
	var out []TimeBasedCCMessage
	for _, e := range arr {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		ts, _ := m["timestamp"].(string)
		out = append(out, TimeBasedCCMessage{Type: typ, Timestamp: ts})
	}
	return out
}

func collectCompactableToolUseIDsFromCCArray(arr []interface{}) []string {
	var ids []string
	for _, e := range arr {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		if typ != "assistant" {
			continue
		}
		msg, ok := m["message"].(map[string]interface{})
		if !ok {
			continue
		}
		raw, ok := msg["content"]
		if !ok {
			continue
		}
		blocks, ok := raw.([]interface{})
		if !ok {
			continue
		}
		for _, b := range blocks {
			bm, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			bt, _ := bm["type"].(string)
			if bt != "tool_use" {
				continue
			}
			id, _ := bm["id"].(string)
			name, _ := bm["name"].(string)
			if id == "" || !IsCompactableToolName(name) {
				continue
			}
			ids = append(ids, id)
		}
	}
	return ids
}

func mutateUserToolResultsInCCArray(arr []interface{}, clearSet map[string]struct{}) int {
	tokensSaved := 0
	for _, e := range arr {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		if typ != "user" {
			continue
		}
		msg, ok := m["message"].(map[string]interface{})
		if !ok {
			continue
		}
		raw, ok := msg["content"]
		if !ok {
			continue
		}
		blocks, ok := raw.([]interface{})
		if !ok {
			continue
		}
		for i, b := range blocks {
			bm, ok := b.(map[string]interface{})
			if !ok {
				continue
			}
			bt, _ := bm["type"].(string)
			if bt != "tool_result" {
				continue
			}
			tuid, _ := bm["tool_use_id"].(string)
			if tuid == "" {
				continue
			}
			if _, want := clearSet[tuid]; !want {
				continue
			}
			if s, ok := bm["content"].(string); ok && s == TimeBasedMCClearedMessage {
				continue
			}
			tokensSaved += toolResultContentTokens(bm["content"])
			bm["content"] = TimeBasedMCClearedMessage
			blocks[i] = bm
		}
		msg["content"] = blocks
	}
	return tokensSaved
}

func toolResultContentTokens(v interface{}) int {
	switch x := v.(type) {
	case nil:
		return 0
	case string:
		return roughTokenEstimation(x)
	case []interface{}:
		sum := 0
		for _, it := range x {
			im, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			typ, _ := im["type"].(string)
			switch typ {
			case "text":
				if s, ok := im["text"].(string); ok {
					sum += roughTokenEstimation(s)
				}
			case "image", "document":
				sum += imageMaxTokenSizeMicrocompact
			default:
				if b, err := json.Marshal(im); err == nil {
					sum += roughTokenEstimation(string(b))
				}
			}
		}
		return sum
	default:
		if b, err := json.Marshal(x); err == nil {
			return roughTokenEstimation(string(b))
		}
		return 0
	}
}

func roughTokenEstimation(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4
}
