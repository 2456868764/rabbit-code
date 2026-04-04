package query

import (
	"errors"
	"fmt"
	"testing"
)

func TestOrphanToolUseID(t *testing.T) {
	err := &OrphanPermissionError{ToolUseID: "tu_1"}
	if !errors.Is(err, ErrOrphanPermission) {
		t.Fatal("expected Is orphan")
	}
	wrapped := fmt.Errorf("wrap: %w", err)
	if !errors.Is(wrapped, ErrOrphanPermission) {
		t.Fatal("expected Is on wrap")
	}
	id, ok := OrphanToolUseID(err)
	if !ok || id != "tu_1" {
		t.Fatalf("got %q %v", id, ok)
	}
	id, ok = OrphanToolUseID(errors.New("other"))
	if ok || id != "" {
		t.Fatalf("unexpected %q %v", id, ok)
	}
}
