package query

import (
	"errors"
	"fmt"
)

// ErrOrphanPermission marks a tool failure caused by a permission / orphan tool_use (P5.3.3 bridge for Phase 6).
var ErrOrphanPermission = errors.New("query: orphan permission")

// OrphanPermissionError wraps ErrOrphanPermission with the tool_use id.
type OrphanPermissionError struct {
	ToolUseID string
}

func (e *OrphanPermissionError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("query: orphan permission: id=%s", e.ToolUseID)
}

func (e *OrphanPermissionError) Unwrap() error { return ErrOrphanPermission }

// OrphanToolUseID returns (id, true) if err is or wraps *OrphanPermissionError.
func OrphanToolUseID(err error) (id string, ok bool) {
	var o *OrphanPermissionError
	if errors.As(err, &o) && o != nil {
		return o.ToolUseID, true
	}
	return "", false
}
