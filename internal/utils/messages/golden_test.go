package messages

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGolden_transcript_v1_minimal(t *testing.T) {
	b := readGolden(t, "transcript_v1_minimal.json")
	tr, err := ParseTranscriptJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if tr.TranscriptVersion != 1 || len(tr.Messages) != 2 {
		t.Fatalf("%+v", tr)
	}
	if tr.Messages[0].Role != "user" || tr.Messages[1].Role != "assistant" {
		t.Fatal(tr.Messages)
	}
	out, err := CanonicalJSON(tr)
	if err != nil {
		t.Fatal(err)
	}
	tr2, err := ParseTranscriptJSON(out)
	if err != nil {
		t.Fatal(err)
	}
	h1, _ := SHA256Hex(tr)
	h2, _ := SHA256Hex(tr2)
	if h1 != h2 {
		t.Fatalf("hash drift %s vs %s", h1, h2)
	}
}

func TestGolden_transcript_tool_chain_strict(t *testing.T) {
	b := readGolden(t, "transcript_tool_chain.json")
	tr, err := ParseTranscriptJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateToolPairing(tr.Messages, true); err != nil {
		t.Fatal(err)
	}
}

func readGolden(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	dir := filepath.Dir(file)
	path := filepath.Join(dir, "testdata", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
