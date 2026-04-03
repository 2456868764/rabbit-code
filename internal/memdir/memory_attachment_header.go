package memdir

import (
	"fmt"
	"time"
)

// MemoryAttachmentHeaderAt mirrors attachments.ts memoryHeader(path, mtimeMs).
func MemoryAttachmentHeaderAt(path string, mtimeMs int64, now time.Time) string {
	staleness := MemoryFreshnessTextAt(mtimeMs, now)
	if staleness != "" {
		return fmt.Sprintf("%s\n\nMemory: %s:", staleness, path)
	}
	return fmt.Sprintf("Memory (saved %s): %s:", MemoryAgeAt(mtimeMs, now), path)
}

// MemoryAttachmentHeader uses time.Now().
func MemoryAttachmentHeader(path string, mtimeMs int64) string {
	return MemoryAttachmentHeaderAt(path, mtimeMs, time.Now())
}
