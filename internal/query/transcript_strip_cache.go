package query

import (
	"bytes"
	"encoding/json"
	"errors"
)

// StripCacheControlFromMessagesJSON returns a copy of the messages array JSON with every
// "cache_control" key removed recursively (mirrors stripCacheControl in promptCacheBreakDetection.ts).
// Used after ErrPromptCacheBreakDetected to force a fresh prompt without stale cache breakpoints.
// changed is true only if at least one cache_control key was removed (ignores JSON re-encoding noise).
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
