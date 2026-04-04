// JavaScript string length (UTF-16 code units) parity for notebook / Bash formatOutput.
package messages

import "unicode/utf8"

// jsStringUTF16Len mirrors String.prototype.length (surrogate pairs count as 2).
func jsStringUTF16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xffff {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// truncateJSStringToMaxUTF16 returns prefix (<= maxUnits UTF-16 code units) and suffix (rest of s).
func truncateJSStringToMaxUTF16(s string, maxUnits int) (prefix, suffix string) {
	if maxUnits <= 0 {
		return "", s
	}
	units := 0
	i := 0
	for i < len(s) {
		r, sz := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && sz == 1 {
			i++
			continue
		}
		add := 1
		if r > 0xffff {
			add = 2
		}
		if units+add > maxUnits {
			return s[:i], s[i:]
		}
		units += add
		i += sz
	}
	return s, ""
}
