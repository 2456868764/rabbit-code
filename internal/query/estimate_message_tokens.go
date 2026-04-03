package query

import "encoding/json"

// ImageDocumentTokenEstimate mirrors microCompact.ts IMAGE_MAX_TOKEN_SIZE (2000).
const ImageDocumentTokenEstimate = 2000

// EstimateMessageTokensFromTranscriptJSON mirrors microCompact.ts estimateMessageTokens for API-shaped
// messages JSON ([{role, content}, ...]); pads by ceil(4/3) like TS.
func EstimateMessageTokensFromTranscriptJSON(transcript []byte) (int, error) {
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return 0, err
	}
	total := 0
	for _, m := range arr {
		role := jsonStringField(m["role"])
		if role != "user" && role != "assistant" {
			continue
		}
		c := m["content"]
		if len(c) == 0 {
			continue
		}
		switch c[0] {
		case '"':
			var s string
			if err := json.Unmarshal(c, &s); err == nil {
				total += EstimateUTF8BytesAsTokens(s)
			}
		case '[':
			var blocks []map[string]json.RawMessage
			if err := json.Unmarshal(c, &blocks); err != nil {
				continue
			}
			for _, b := range blocks {
				typ := jsonStringField(b["type"])
				switch typ {
				case "text":
					total += EstimateUTF8BytesAsTokens(jsonStringField(b["text"]))
				case "tool_result":
					total += estimateToolResultContentTokens(b["content"])
				case "image", "document":
					total += ImageDocumentTokenEstimate
				case "thinking":
					total += EstimateUTF8BytesAsTokens(jsonStringField(b["thinking"]))
				case "redacted_thinking":
					total += EstimateUTF8BytesAsTokens(jsonStringField(b["data"]))
				case "tool_use":
					name := jsonStringField(b["name"])
					in := ""
					if raw, ok := b["input"]; ok && len(raw) > 0 {
						in = string(raw)
					}
					total += EstimateUTF8BytesAsTokens(name + in)
				default:
					total += EstimateUTF8BytesAsTokens(string(jsonBlockStringify(b)))
				}
			}
		}
	}
	if total == 0 {
		return 0, nil
	}
	return (total*4 + 2) / 3, nil
}

func jsonStringField(raw json.RawMessage) string {
	if len(raw) == 0 || raw[0] != '"' {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

func estimateToolResultContentTokens(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return EstimateUTF8BytesAsTokens(s)
		}
		return 0
	}
	var arr []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return EstimateUTF8BytesAsTokens(string(raw))
	}
	sum := 0
	for _, it := range arr {
		switch it.Type {
		case "text":
			sum += EstimateUTF8BytesAsTokens(it.Text)
		case "image", "document":
			sum += ImageDocumentTokenEstimate
		default:
			sum += EstimateUTF8BytesAsTokens(it.Type)
		}
	}
	return sum
}

func jsonBlockStringify(b map[string]json.RawMessage) string {
	out, err := json.Marshal(b)
	if err != nil {
		return ""
	}
	return string(out)
}
