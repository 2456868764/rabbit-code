package bashtool

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SplitCommandWithOperators mirrors restored-src/src/utils/bash/commands.ts splitCommandWithOperators
// for typical ASCII shell input: continuation join, heredoc fallback to a single segment, balanced-quote
// gate, then tokenization. Differs from TS on complex shell-quote/heredoc edge cases (falls back to one segment).
func SplitCommandWithOperators(command string) []string {
	joined := joinShellContinuations(command)
	if strings.Contains(command, "<<") {
		return []string{joined}
	}
	if !shellQuotesBalanced(joined) {
		return []string{joined}
	}
	toks := tokenizeShell(joined)
	if len(toks) == 0 {
		return nil
	}
	return toks
}

// SplitCommandDeprecated mirrors splitCommand_DEPRECATED: operators split, redirections stripped, control operators removed.
func SplitCommandDeprecated(command string) []string {
	parts := SplitCommandWithOperators(command)
	if len(parts) == 0 {
		return nil
	}
	parts = stripRedirectionTokens(append([]string(nil), parts...))
	return filterControlOperators(parts)
}

var backslashNewlineRE = regexp.MustCompile(`\\+\n`)

func joinShellContinuations(s string) string {
	return backslashNewlineRE.ReplaceAllStringFunc(s, func(m string) string {
		n := len(m) - 1 // backslashes before newline
		if n%2 == 1 {
			return strings.Repeat(`\`, n-1)
		}
		return m
	})
}

func shellQuotesBalanced(s string) bool {
	inS, inD := false, false
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		i += w
		if inS {
			if r == '\'' {
				inS = false
			}
			continue
		}
		if inD {
			if r == '\\' && i < len(s) {
				_, w2 := utf8.DecodeRuneInString(s[i:])
				i += w2
				continue
			}
			if r == '"' {
				inD = false
			}
			continue
		}
		switch r {
		case '\'':
			inS = true
		case '"':
			inD = true
		}
	}
	return !inS && !inD
}

func tokenizeShell(s string) []string {
	var tokens []string
	var b strings.Builder
	rs := []rune(s)
	inS, inD := false, false
	i := 0
	flush := func() {
		t := strings.TrimSpace(b.String())
		b.Reset()
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	for i < len(rs) {
		r := rs[i]
		if inS {
			b.WriteRune(r)
			if r == '\'' {
				inS = false
			}
			i++
			continue
		}
		if inD {
			if r == '\\' && i+1 < len(rs) {
				b.WriteRune(r)
				b.WriteRune(rs[i+1])
				i += 2
				continue
			}
			b.WriteRune(r)
			if r == '"' {
				inD = false
			}
			i++
			continue
		}
		if r == '\'' {
			flush()
			inS = true
			b.WriteRune(r)
			i++
			continue
		}
		if r == '"' {
			flush()
			inD = true
			b.WriteRune(r)
			i++
			continue
		}
		if r == '\n' || r == '\r' {
			flush()
			if r == '\r' && i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
			tokens = append(tokens, ";")
			i++
			continue
		}
		if unicode.IsSpace(r) {
			if b.Len() == 0 {
				i++
				continue
			}
			b.WriteRune(r)
			i++
			continue
		}
		rest := string(rs[i:])
		switch {
		case strings.HasPrefix(rest, "&&"):
			flush()
			tokens = append(tokens, "&&")
			i += 2
			continue
		case strings.HasPrefix(rest, "||"):
			flush()
			tokens = append(tokens, "||")
			i += 2
			continue
		case strings.HasPrefix(rest, ">>"):
			flush()
			tokens = append(tokens, ">>")
			i += 2
			continue
		case strings.HasPrefix(rest, ">&"):
			flush()
			tokens = append(tokens, ">&")
			i += 2
			continue
		case strings.HasPrefix(rest, "|"):
			flush()
			tokens = append(tokens, "|")
			i++
			continue
		case r == ';':
			flush()
			tokens = append(tokens, ";")
			i++
			continue
		case r == '>':
			flush()
			tokens = append(tokens, ">")
			i++
			continue
		}
		b.WriteRune(r)
		i++
	}
	flush()
	return tokens
}

func allowedFileDescriptor(ch byte) bool {
	return ch == '0' || ch == '1' || ch == '2'
}

func isStaticRedirectTarget(target string) bool {
	if target == "" || strings.ContainsAny(target, " \t\n\r'\"") {
		return false
	}
	if strings.HasPrefix(target, "#") {
		return false
	}
	if strings.HasPrefix(target, "!") || strings.HasPrefix(target, "=") {
		return false
	}
	for i := 0; i < len(target); i++ {
		switch target[i] {
		case '$', '`', '*', '?', '[', '{', '~', '(', '<':
			return false
		}
	}
	if strings.HasPrefix(target, "&") {
		return false
	}
	return true
}

func stripRedirectionTokens(parts []string) []string {
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		if part != ">&" && part != ">" && part != ">>" {
			continue
		}
		prevPart := ""
		if i > 0 {
			prevPart = strings.TrimSpace(parts[i-1])
		}
		nextPart := ""
		if i+1 < len(parts) {
			nextPart = strings.TrimSpace(parts[i+1])
		}
		afterNext := ""
		if i+2 < len(parts) {
			afterNext = strings.TrimSpace(parts[i+2])
		}
		if nextPart == "" {
			continue
		}
		effectiveNext := nextPart
		if (part == ">" || part == ">>") && len(nextPart) >= 3 &&
			nextPart[len(nextPart)-2] == ' ' &&
			allowedFileDescriptor(nextPart[len(nextPart)-1]) &&
			(afterNext == ">" || afterNext == ">>" || afterNext == ">&") {
			effectiveNext = nextPart[:len(nextPart)-2]
		}
		shouldStrip := false
		stripThird := false
		switch {
		case part == ">&" && len(nextPart) == 1 && allowedFileDescriptor(nextPart[0]):
			shouldStrip = true
		case part == ">" && nextPart == "&" && afterNext != "" && len(afterNext) == 1 && allowedFileDescriptor(afterNext[0]):
			shouldStrip = true
			stripThird = true
		case part == ">" && strings.HasPrefix(nextPart, "&") && len(nextPart) > 1 && allowedFileDescriptor(nextPart[1]):
			shouldStrip = true
		case (part == ">" || part == ">>") && isStaticRedirectTarget(effectiveNext):
			shouldStrip = true
		}
		if !shouldStrip {
			continue
		}
		if prevPart != "" && len(prevPart) >= 3 &&
			allowedFileDescriptor(prevPart[len(prevPart)-1]) &&
			prevPart[len(prevPart)-2] == ' ' {
			parts[i-1] = prevPart[:len(prevPart)-2]
		}
		parts[i] = ""
		parts[i+1] = ""
		if stripThird && i+2 < len(parts) {
			parts[i+2] = ""
		}
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

var controlOperators = map[string]struct{}{
	"&&": {},
	"||": {},
	"|":  {},
	";":  {},
}

func filterControlOperators(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if _, ok := controlOperators[p]; !ok {
			out = append(out, p)
		}
	}
	return out
}
