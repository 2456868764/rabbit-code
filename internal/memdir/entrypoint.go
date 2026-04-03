package memdir

import (
	"fmt"
	"strings"
)

// Entrypoint file name for memory index (memdir.ts ENTRYPOINT_NAME).
const EntrypointName = "MEMORY.md"

// MaxEntrypointLines caps MEMORY.md line count before truncation (memdir.ts).
const MaxEntrypointLines = 200

// MaxEntrypointBytes caps MEMORY.md UTF-8 size before truncation (memdir.ts).
const MaxEntrypointBytes = 25_000

// EntrypointTruncation is the result of TruncateEntrypointContent (memdir.ts).
type EntrypointTruncation struct {
	Content          string
	LineCount        int
	ByteCount        int
	WasLineTruncated bool
	WasByteTruncated bool
}

// TruncateEntrypointContent applies line then byte caps and appends a warning (memdir.ts truncateEntrypointContent).
func TruncateEntrypointContent(raw string) EntrypointTruncation {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return EntrypointTruncation{Content: "", LineCount: 0, ByteCount: 0}
	}
	contentLines := strings.Split(trimmed, "\n")
	lineCount := len(contentLines)
	byteCount := len(trimmed)

	wasLineTruncated := lineCount > MaxEntrypointLines
	wasByteTruncated := byteCount > MaxEntrypointBytes

	if !wasLineTruncated && !wasByteTruncated {
		return EntrypointTruncation{
			Content:          trimmed,
			LineCount:        lineCount,
			ByteCount:        byteCount,
			WasLineTruncated: false,
			WasByteTruncated: false,
		}
	}

	truncated := trimmed
	if wasLineTruncated {
		truncated = strings.Join(contentLines[:MaxEntrypointLines], "\n")
	}

	if len(truncated) > MaxEntrypointBytes {
		search := truncated
		if len(search) > MaxEntrypointBytes {
			search = search[:MaxEntrypointBytes]
		}
		cutAt := strings.LastIndex(search, "\n")
		if cutAt <= 0 {
			truncated = truncated[:MaxEntrypointBytes]
		} else {
			truncated = truncated[:cutAt]
		}
	}

	var reason string
	switch {
	case wasByteTruncated && !wasLineTruncated:
		reason = fmt.Sprintf("%d bytes (limit: %d) — index entries are too long", byteCount, MaxEntrypointBytes)
	case wasLineTruncated && !wasByteTruncated:
		reason = fmt.Sprintf("%d lines (limit: %d)", lineCount, MaxEntrypointLines)
	default:
		reason = fmt.Sprintf("%d lines and %d bytes", lineCount, byteCount)
	}

	warning := fmt.Sprintf(
		"\n\n> WARNING: %s is %s. Only part of it was loaded. Keep index entries to one line under ~200 chars; move detail into topic files.",
		EntrypointName, reason,
	)

	return EntrypointTruncation{
		Content:          truncated + warning,
		LineCount:        lineCount,
		ByteCount:        byteCount,
		WasLineTruncated: wasLineTruncated,
		WasByteTruncated: wasByteTruncated,
	}
}
