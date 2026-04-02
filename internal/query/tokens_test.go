package query

import "testing"

func TestEstimateUTF8BytesAsTokens(t *testing.T) {
	if EstimateUTF8BytesAsTokens("") != 0 {
		t.Fatal()
	}
	if EstimateUTF8BytesAsTokens("abcd") != 1 {
		t.Fatalf("got %d", EstimateUTF8BytesAsTokens("abcd"))
	}
	if EstimateUTF8BytesAsTokens("abcde") != 2 {
		t.Fatalf("got %d", EstimateUTF8BytesAsTokens("abcde"))
	}
}

func TestEstimateTranscriptJSONTokens(t *testing.T) {
	if EstimateTranscriptJSONTokens([]byte(`{"x":1}`)) != 2 {
		t.Fatalf("got %d", EstimateTranscriptJSONTokens([]byte(`{"x":1}`)))
	}
}
