package anthropic

import "testing"

func TestParsePromptTooLongTokenCounts(t *testing.T) {
	a, l, ok := ParsePromptTooLongTokenCounts("Prompt is too long: 137500 tokens > 135000 maximum")
	if !ok || a != 137500 || l != 135000 {
		t.Fatalf("%d %d %v", a, l, ok)
	}
}

func TestClassifyBody(t *testing.T) {
	if ClassifyBody("x") != KindUnknown {
		t.Fatal()
	}
	if ClassifyBody("Prompt is too long") != KindPromptTooLong {
		t.Fatal()
	}
}
