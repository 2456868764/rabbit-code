package memdir

import (
	"fmt"
	"strings"
	"time"
)

// FormatMemoryManifest mirrors memoryScan.ts formatMemoryManifest (one line per memory).
func FormatMemoryManifest(memories []MemoryHeader) string {
	var b strings.Builder
	for i, m := range memories {
		if i > 0 {
			b.WriteByte('\n')
		}
		tag := ""
		if m.Type != "" {
			tag = fmt.Sprintf("[%s] ", m.Type)
		}
		ts := time.UnixMilli(m.MtimeMs).UTC().Format(time.RFC3339)
		line := fmt.Sprintf("- %s%s (%s)", tag, m.Filename, ts)
		if m.Description != "" {
			line += ": " + m.Description
		}
		b.WriteString(line)
	}
	return b.String()
}
