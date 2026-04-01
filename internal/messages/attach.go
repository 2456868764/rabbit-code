package messages

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/2456868764/rabbit-code/internal/types"
)

// VerifyFileRefs checks each file_ref content piece: Ref path readable and optional Sha256 matches (P3.3.2).
func VerifyFileRefs(msgs []types.Message) error {
	for mi, m := range msgs {
		for ci, c := range m.Content {
			if c.Type != types.BlockTypeFileRef {
				continue
			}
			if c.Ref == "" {
				return fmt.Errorf("message %d content %d: file_ref missing ref", mi, ci)
			}
			b, err := os.ReadFile(c.Ref)
			if err != nil {
				return fmt.Errorf("message %d content %d: file_ref %q: %w", mi, ci, c.Ref, err)
			}
			if c.Sha256 != "" {
				sum := sha256.Sum256(b)
				got := hex.EncodeToString(sum[:])
				if got != c.Sha256 {
					return fmt.Errorf("message %d content %d: file_ref %q sha256 want %s got %s", mi, ci, c.Ref, c.Sha256, got)
				}
			}
		}
	}
	return nil
}
