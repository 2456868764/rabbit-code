package query

import "testing"

func TestEstimateMessageTokensFromTranscriptJSON_textAndToolUse(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":[{"type":"text","text":"abcd"}]},
		{"role":"assistant","content":[{"type":"tool_use","id":"x","name":"Read","input":{"p":1}}]}
	]`)
	n, err := EstimateMessageTokensFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected positive tokens, got %d", n)
	}
	base := EstimateUTF8BytesAsTokens("abcd") + EstimateUTF8BytesAsTokens(`Read{"p":1}`)
	want := (base*4 + 2) / 3
	if n != want {
		t.Fatalf("got %d want %d", n, want)
	}
}

func TestEstimateMessageTokensFromTranscriptJSON_invalid(t *testing.T) {
	_, err := EstimateMessageTokensFromTranscriptJSON([]byte(`not-json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEstimateMessageTokensFromTranscriptJSON_imageBase64Heuristic(t *testing.T) {
	longB64 := make([]byte, 4000)
	for i := range longB64 {
		longB64[i] = 'A'
	}
	raw := []byte(`[{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"` + string(longB64) + `"}}]}]`)
	n, err := EstimateMessageTokensFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Should exceed flat 2000-token image cap when base64 is large.
	if n <= 2000 {
		t.Fatalf("expected large base64 to raise estimate, got %d", n)
	}
}
