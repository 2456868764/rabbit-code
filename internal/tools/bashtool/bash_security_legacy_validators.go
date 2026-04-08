package bashtool

import (
	"regexp"
	"strings"
)

// Subset of bashSecurity.ts bashCommandIsSafe_DEPRECATED validators for BashReadOnlySecurityRejectReason.

var (
	reIFSInjection        = regexp.MustCompile(`\$IFS|\$\{[^}]*IFS`)
	reProcEnviron         = regexp.MustCompile(`/proc/.*/environ`)
	reUnicodeWS           = regexp.MustCompile(`[\x{00A0}\x{1680}\x{2000}-\x{200A}\x{2028}\x{2029}\x{202F}\x{205F}\x{3000}\x{FEFF}]`)
	reObfuscatedANSI      = regexp.MustCompile(`\$'[^']*'`)
	reObfuscatedLocale    = regexp.MustCompile(`\$"[^"]*"`)
	reObfuscatedEmptyDash = regexp.MustCompile(`\$['"]{2}\s*-`)
	reJqSystem            = regexp.MustCompile(`(?i)\bsystem\s*\(`)
	reJqDangerousFlags    = regexp.MustCompile(`(?:^|\s)(?:-f\b|--from-file|--rawfile|--slurpfile|-L\b|--library-path)`)
	reJqLineLead = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_]*=\S*\s+)*jq\b`)
	reDangerVarPipe       = regexp.MustCompile(`[<>|]\s*\$[A-Za-z_]`)
	reDangerVarPipe2      = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*\s*[|<>]`)
	reQuotedBraceObfus = regexp.MustCompile(`['"][{}]['"]`)
	// Go regexp has no lookahead; use (?:\s|$) instead of (?=\s|$) from TS stripSafeRedirections.
	strip2Then1     = regexp.MustCompile(`\s+2\s*>&\s*1(?:\s|$)`)
	stripDevNullOut = regexp.MustCompile(`[012]?\s*>\s*/dev/null(?:\s|$)`)
	stripDevNullIn  = regexp.MustCompile(`\s*<\s*/dev/null(?:\s|$)`)
)

func bashCarriageReturnRejectReason(command string) string {
	if !strings.ContainsRune(command, '\r') {
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
		if c == '\r' && !inD {
			return "command contains carriage return (\\r) which shell-quote and bash tokenize differently"
		}
	}
	return ""
}

func bashIFSRejectReason(command string) string {
	if reIFSInjection.MatchString(command) {
		return "command contains IFS variable usage which could bypass security validation"
	}
	return ""
}

func bashProcEnvironRejectReason(command string) string {
	if reProcEnviron.MatchString(command) {
		return "command accesses /proc/*/environ which could expose sensitive environment variables"
	}
	return ""
}

func bashUnicodeWhitespaceRejectReason(command string) string {
	if reUnicodeWS.MatchString(command) {
		return "command contains Unicode whitespace characters that could cause parsing inconsistencies"
	}
	return ""
}

func bashObfuscatedQuotingRejectReason(command string) string {
	switch {
	case reObfuscatedANSI.MatchString(command):
		return "command contains ANSI-C quoting ($'...') which can hide characters"
	case reObfuscatedLocale.MatchString(command):
		return "command contains locale quoting ($\"...\") which can hide characters"
	case reObfuscatedEmptyDash.MatchString(command):
		return "command contains empty special quotes before dash (potential bypass)"
	}
	return ""
}

// bashFullyUnquotedForBrace mirrors bashSecurity extractQuotedContent.fullyUnquoted (non-jq).
func bashFullyUnquotedForBrace(command string) string {
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
			continue
		}
		if c == '"' && !inS {
			inD = !inD
			continue
		}
		if !inS && !inD {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func stripSafeRedirectionsGo(content string) string {
	s := strip2Then1.ReplaceAllString(content, "")
	s = stripDevNullOut.ReplaceAllString(s, "")
	s = stripDevNullIn.ReplaceAllString(s, "")
	return s
}

func bashDangerousVarRejectReason(command string) string {
	fu := stripSafeRedirectionsGo(bashFullyUnquotedForBrace(command))
	if reDangerVarPipe.MatchString(fu) || reDangerVarPipe2.MatchString(fu) {
		return "command contains variables in dangerous contexts (redirections or pipes)"
	}
	return ""
}

func isEscapedAtPosition(content string, pos int) bool {
	n := 0
	for i := pos - 1; i >= 0 && content[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}

func bashBraceExpansionRejectReason(originalCommand string) string {
	content := bashFullyUnquotedForBrace(originalCommand)

	openB, closeB := 0, 0
	for i := 0; i < len(content); i++ {
		switch content[i] {
		case '{':
			if !isEscapedAtPosition(content, i) {
				openB++
			}
		case '}':
			if !isEscapedAtPosition(content, i) {
				closeB++
			}
		}
	}
	if openB > 0 && closeB > openB {
		return "command has excess closing braces after quote stripping, indicating possible brace expansion obfuscation"
	}
	if openB > 0 && reQuotedBraceObfus.MatchString(originalCommand) {
		return "command contains quoted brace character inside brace context (potential brace expansion obfuscation)"
	}

	for i := 0; i < len(content); i++ {
		if content[i] != '{' || isEscapedAtPosition(content, i) {
			continue
		}
		depth := 1
		matchingClose := -1
		for j := i + 1; j < len(content); j++ {
			ch := content[j]
			if ch == '{' && !isEscapedAtPosition(content, j) {
				depth++
			} else if ch == '}' && !isEscapedAtPosition(content, j) {
				depth--
				if depth == 0 {
					matchingClose = j
					break
				}
			}
		}
		if matchingClose < 0 {
			continue
		}
		innerDepth := 0
		for k := i + 1; k < matchingClose; k++ {
			ch := content[k]
			if ch == '{' && !isEscapedAtPosition(content, k) {
				innerDepth++
			} else if ch == '}' && !isEscapedAtPosition(content, k) {
				innerDepth--
			} else if innerDepth == 0 {
				if ch == ',' || (ch == '.' && k+1 < matchingClose && content[k+1] == '.') {
					return "command contains brace expansion that could alter command parsing"
				}
			}
		}
	}
	return ""
}

func bashJqCompoundRejectReason(command string) string {
	for _, seg := range shellSegments(command) {
		for _, sub := range SplitCommandDeprecated(seg) {
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			if r := bashJqLineRejectReason(sub); r != "" {
				return r
			}
		}
	}
	return ""
}

func bashJqLineRejectReason(sub string) string {
	if !reJqLineLead.MatchString(sub) {
		return ""
	}
	if reJqSystem.MatchString(sub) {
		return "jq command contains system() function which executes arbitrary commands"
	}
	loc := reJqLineLead.FindStringIndex(sub)
	if loc == nil {
		return ""
	}
	after := strings.TrimSpace(sub[loc[1]:])
	if reJqDangerousFlags.MatchString(after) {
		return "jq command contains dangerous flags that could execute code or read arbitrary files"
	}
	return ""
}
