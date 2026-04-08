package bashtool

import (
	"regexp"
	"strings"
	"unicode"
)

// Remaining bashCommandIsSafe_DEPRECATED validators (bashSecurity.ts).

var (
	reIncompleteOpLead = regexp.MustCompile(`^\s*(&&|\|\||;|>>?|<)`)
	reIncompleteTab    = regexp.MustCompile(`^\s*\t`)
	reBackslashNL      = regexp.MustCompile(`\\+\n`)
	reShellMetaQuoted  = regexp.MustCompile(`(?:^|\s)["'][^"']*[;&][^"']*["'](?:\s|$)`)
	reFindNameMeta     = regexp.MustCompile(`-name\s+["'][^"']*[;|&][^"']*["']`)
	reFindPathMeta     = regexp.MustCompile(`-path\s+["'][^"']*[;|&][^"']*["']`)
	reFindInameMeta    = regexp.MustCompile(`-iname\s+["'][^"']*[;|&][^"']*["']`)
	reFindRegexMeta    = regexp.MustCompile(`-regex\s+["'][^"']*[;&][^"']*["']`)
)

func bashIncompleteCommandsRejectReason(command string) string {
	if reIncompleteTab.MatchString(command) {
		return "command appears to be an incomplete fragment (starts with tab)"
	}
	if t := strings.TrimSpace(command); t != "" && strings.HasPrefix(t, "-") {
		return "command appears to be an incomplete fragment (starts with flags)"
	}
	if reIncompleteOpLead.MatchString(command) {
		return "command appears to be a continuation line (starts with operator)"
	}
	return ""
}

func bashUnquotedKeepQuoteChars(command string) string {
	var b strings.Builder
	inS, inD := false, false
	escaped := false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if escaped {
			escaped = false
			if !inS && !inD {
				b.WriteByte(c)
			}
			continue
		}
		if c == '\\' && !inS {
			escaped = true
			if !inS && !inD {
				b.WriteByte(c)
			}
			continue
		}
		if c == '\'' && !inD {
			inS = !inS
			b.WriteByte(c)
			continue
		}
		if c == '"' && !inS {
			inD = !inD
			b.WriteByte(c)
			continue
		}
		if !inS && !inD {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func bashJoinBackslashNewlinesForHash(s string) string {
	return reBackslashNL.ReplaceAllStringFunc(s, func(match string) string {
		bs := len(match) - 1
		if bs%2 == 1 {
			return strings.Repeat(`\`, bs-1)
		}
		return match
	})
}

func isShellWhitespaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func bashMidWordHashIn(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != '#' {
			continue
		}
		if isShellWhitespaceByte(s[i-1]) {
			continue
		}
		if i >= 2 && s[i-2] == '$' && s[i-1] == '{' {
			continue
		}
		return true
	}
	return false
}

func bashMidWordHashRejectReason(command string) string {
	keep := bashUnquotedKeepQuoteChars(command)
	joined := bashJoinBackslashNewlinesForHash(keep)
	if bashMidWordHashIn(keep) || bashMidWordHashIn(joined) {
		return "command contains mid-word # which is parsed differently by shell-quote vs bash"
	}
	return ""
}

func bashCommentQuoteDesyncRejectReason(command string) string {
	inS, inD := false, false
	escaped := false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if escaped {
			escaped = false
			continue
		}
		if inS {
			if c == '\'' {
				inS = false
			}
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if inD {
			if c == '"' {
				inD = false
			}
			continue
		}
		if c == '\'' {
			inS = true
			continue
		}
		if c == '"' {
			inD = true
			continue
		}
		if c == '#' {
			rest := command[i+1:]
			lineEnd := strings.IndexByte(rest, '\n')
			var comment string
			if lineEnd < 0 {
				comment = rest
			} else {
				comment = rest[:lineEnd]
			}
			if strings.ContainsAny(comment, `'"`) {
				return "command contains quote characters inside a # comment which can desync quote tracking"
			}
			if lineEnd < 0 {
				break
			}
			i += 1 + lineEnd
		}
	}
	return ""
}

func bashQuotedNewlineRejectReason(command string) string {
	if !strings.ContainsRune(command, '\n') || !strings.ContainsRune(command, '#') {
		return ""
	}
	inS, inD := false, false
	escaped := false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && !inS {
			escaped = true
			continue
		}
		if c == '\'' && !inD {
			inS = !inS
			continue
		}
		if c == '"' && !inS {
			inD = !inD
			continue
		}
		if c == '\n' && (inS || inD) {
			lineStart := i + 1
			nextNL := strings.IndexByte(command[lineStart:], '\n')
			var lineEnd int
			if nextNL < 0 {
				lineEnd = len(command)
			} else {
				lineEnd = lineStart + nextNL
			}
			nextLine := command[lineStart:lineEnd]
			trimmed := strings.TrimFunc(nextLine, unicode.IsSpace)
			if strings.HasPrefix(trimmed, "#") {
				return "command contains a quoted newline followed by a #-prefixed line, which can hide arguments from line-based permission checks"
			}
		}
	}
	return ""
}

func bashNewlinesRejectReason(command string) string {
	fu := bashFullyUnquotedForBrace(command)
	if !strings.ContainsAny(fu, "\n\r") {
		return ""
	}
	for i := 0; i < len(fu); i++ {
		if fu[i] != '\n' && fu[i] != '\r' {
			continue
		}
		j := i + 1
		for j < len(fu) && (fu[j] == ' ' || fu[j] == '\t') {
			j++
		}
		if j >= len(fu) {
			continue
		}
		if fu[j] == ' ' || fu[j] == '\t' || fu[j] == '\n' || fu[j] == '\r' {
			continue
		}
		if i > 0 && fu[i-1] == '\\' && i >= 2 && (fu[i-2] == ' ' || fu[i-2] == '\t') {
			continue
		}
		return "command contains newlines that could separate multiple commands"
	}
	return ""
}

func bashRedirectionsRejectReason(command string) string {
	fu := stripSafeRedirectionsGo(bashFullyUnquotedForBrace(command))
	if strings.ContainsRune(fu, '<') {
		return "command contains input redirection (<) which could read sensitive files"
	}
	if strings.ContainsRune(fu, '>') {
		return "command contains output redirection (>) which could write to arbitrary files"
	}
	return ""
}

func bashBackslashEscapedWhitespaceRejectReason(command string) string {
	inS, inD := false, false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if c == '\\' && !inS {
			if !inD {
				if i+1 < len(command) {
					n := command[i+1]
					if n == ' ' || n == '\t' {
						return "command contains backslash-escaped whitespace that could alter command parsing"
					}
				}
			}
			i++
			continue
		}
		if c == '"' && !inS {
			inD = !inD
			continue
		}
		if c == '\'' && !inD {
			inS = !inS
			continue
		}
	}
	return ""
}

func bashBackslashEscapedOperatorsRejectReason(command string) string {
	const shellOps = ";|&<>"
	inS, inD := false, false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if c == '\\' && !inS {
			if !inD && i+1 < len(command) {
				n := command[i+1]
				if strings.ContainsRune(shellOps, rune(n)) {
					return "command contains a backslash before a shell operator (;, |, &, <, >) which can hide command structure"
				}
			}
			i++
			continue
		}
		if c == '\'' && !inD {
			inS = !inS
			continue
		}
		if c == '"' && !inS {
			inD = !inD
			continue
		}
	}
	return ""
}

func bashShellMetacharactersRejectReason(u string) string {
	msg := "command contains shell metacharacters (;, |, or &) in arguments"
	if reShellMetaQuoted.MatchString(u) {
		return msg
	}
	if reFindNameMeta.MatchString(u) || reFindPathMeta.MatchString(u) || reFindInameMeta.MatchString(u) {
		return msg
	}
	if reFindRegexMeta.MatchString(u) {
		return msg
	}
	return ""
}
