package messages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/2456868764/rabbit-code/internal/types"
)

// CurrentTranscriptVersion is written to new transcript files.
const CurrentTranscriptVersion = 1

// Transcript is a versioned on-disk conversation (session file subset, Phase 3).
type Transcript struct {
	TranscriptVersion int             `json:"transcript_version"`
	Messages          []types.Message `json:"messages"`
}

// CanonicalJSON returns deterministic JSON for hashing and golden tests.
func CanonicalJSON(t *Transcript) ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf("transcript is nil")
	}
	return json.MarshalIndent(t, "", "  ")
}

// SHA256Hex returns hex-encoded SHA-256 of CanonicalJSON.
func SHA256Hex(t *Transcript) (string, error) {
	b, err := CanonicalJSON(t)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ParseTranscriptJSON decodes transcript JSON and sets default version if missing.
func ParseTranscriptJSON(data []byte) (*Transcript, error) {
	var tr Transcript
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, err
	}
	if tr.TranscriptVersion == 0 {
		tr.TranscriptVersion = CurrentTranscriptVersion
	}
	return &tr, nil
}

// ReadTranscriptFile reads and parses a transcript JSON file.
func ReadTranscriptFile(path string) (*Transcript, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseTranscriptJSON(b)
}

// WriteTranscriptFile writes canonical JSON atomically (same directory).
func WriteTranscriptFile(path string, t *Transcript) error {
	if t == nil {
		return fmt.Errorf("transcript is nil")
	}
	if t.TranscriptVersion == 0 {
		t.TranscriptVersion = CurrentTranscriptVersion
	}
	b, err := CanonicalJSON(t)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".transcript-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
