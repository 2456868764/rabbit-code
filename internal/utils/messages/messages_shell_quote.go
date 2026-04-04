// Shell argument quoting parity: TS quote([arg]) from src/utils/bash/shellQuote.ts (npm shell-quote).
package messages

import "strings"

// bashQuoteShellArg returns one Bourne-shell-quoted word suitable for `ls <arg>` (same role as TS quote([path])).
func bashQuoteShellArg(s string) string {
	if s == "" {
		return "''"
	}
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		if r == '\'' {
			b.WriteString(`'"'"'`)
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
