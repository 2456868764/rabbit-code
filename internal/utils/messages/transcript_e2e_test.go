package messages

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/2456868764/rabbit-code/internal/types"
)

// TestE2E_TranscriptRoundTripHash covers AC3-3 (write → read → same hash, count, first role).
func TestE2E_TranscriptRoundTripHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")

	tr := &Transcript{
		TranscriptVersion: CurrentTranscriptVersion,
		Messages: []types.Message{
			{Role: types.RoleUser, Content: []types.ContentPiece{{Type: types.BlockTypeText, Text: "e2e"}}},
			{Role: types.RoleAssistant, Content: []types.ContentPiece{{Type: types.BlockTypeText, Text: "ok"}}},
		},
	}
	h1, err := SHA256Hex(tr)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteTranscriptFile(path, tr); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTranscriptFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("count %d", len(got.Messages))
	}
	if got.Messages[0].Role != types.RoleUser {
		t.Fatal(got.Messages[0].Role)
	}
	h2, err := SHA256Hex(got)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("hash mismatch %s vs %s", h1, h2)
	}
}

func TestVerifyFileRefs_ok(t *testing.T) {
	f := filepath.Join(t.TempDir(), "blob.bin")
	payload := []byte("payload-bytes")
	if err := os.WriteFile(f, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	// sha256 of "payload-bytes" — compute in test
	sum := sha256.Sum256(payload)
	msgs := []types.Message{
		{
			Role: types.RoleUser,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeFileRef, Ref: f, Sha256: hex.EncodeToString(sum[:])},
			},
		},
	}
	if err := VerifyFileRefs(msgs); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFileRefs_badHash(t *testing.T) {
	f := filepath.Join(t.TempDir(), "x")
	_ = os.WriteFile(f, []byte("a"), 0o600)
	msgs := []types.Message{
		{
			Role: types.RoleUser,
			Content: []types.ContentPiece{
				{Type: types.BlockTypeFileRef, Ref: f, Sha256: "00" + "00"},
			},
		},
	}
	if err := VerifyFileRefs(msgs); err == nil {
		t.Fatal("expected error")
	}
}
