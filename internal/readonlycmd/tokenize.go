package readonlycmd

import "errors"

// TokenizeShellWords mirrors tryParseShellCommand string-only success path (no operators).
func TokenizeShellWords(s string) ([]string, error) {
	var toks []string
	var b []byte
	i := 0
	for i < len(s) {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '\'' {
			i++
			start := i
			for i < len(s) && s[i] != '\'' {
				i++
			}
			if i >= len(s) {
				return nil, errors.New("unclosed quote")
			}
			toks = append(toks, s[start:i])
			i++
			continue
		}
		if s[i] == '"' {
			i++
			b = b[:0]
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					b = append(b, s[i+1])
					i += 2
					continue
				}
				if s[i] == '"' {
					break
				}
				b = append(b, s[i])
				i++
			}
			if i >= len(s) {
				return nil, errors.New("unclosed quote")
			}
			toks = append(toks, string(b))
			i++
			continue
		}
		start := i
		for i < len(s) && s[i] != ' ' && s[i] != '\t' {
			i++
		}
		toks = append(toks, s[start:i])
	}
	return toks, nil
}
