// Shell quoting parity with npm shell-quote quote() for a single argument (POSIX sh).
package messages

import "strings"

// ShellQuoteSingleArg mirrors shell-quote's quote() for one string: safe characters
// [A-Za-z0-9_@%+=:,./-] pass through unquoted; otherwise wrap in single quotes,
// escaping embedded ' as '\''. Empty string yields "''".
func ShellQuoteSingleArg(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		if !shellQuoteSafeRune(r) {
			var b strings.Builder
			b.Grow(len(s) + 8)
			b.WriteByte('\'')
			for _, r2 := range s {
				if r2 == '\'' {
					b.WriteString(`'\''`)
				} else {
					b.WriteRune(r2)
				}
			}
			b.WriteByte('\'')
			return b.String()
		}
	}
	return s
}

func shellQuoteSafeRune(r rune) bool {
	if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
		return true
	}
	switch r {
	case '_', '@', '%', '+', '=', ':', ',', '.', '/', '-':
		return true
	default:
		return false
	}
}
