package filereadtool

import (
	"fmt"
	"os"
	"strings"
)

// AddLineNumbers mirrors utils/file.ts addLineNumbers (compact prefix off unless RABBIT_CODE_FILE_READ_COMPACT_LINE_PREFIX=1).
func AddLineNumbers(content string, startLine int) string {
	if content == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n"), "\n")
	compact := os.Getenv("RABBIT_CODE_FILE_READ_COMPACT_LINE_PREFIX") == "1"
	var b strings.Builder
	for i, line := range lines {
		n := startLine + i
		if compact {
			fmt.Fprintf(&b, "%d\t%s", n, line)
		} else {
			ns := fmt.Sprintf("%d", n)
			if len(ns) >= 6 {
				fmt.Fprintf(&b, "%d→%s", n, line)
			} else {
				fmt.Fprintf(&b, "%6d→%s", n, line)
			}
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
