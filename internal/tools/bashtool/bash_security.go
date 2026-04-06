package bashtool

import (
	"regexp"
	"strings"
)

// Mirrors bashSecurity.ts CONTROL_CHAR_RE (excludes tab, LF, CR — handled elsewhere).
var bashControlCharRE = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)

// unquotedOutsideSingleQuotes mirrors bashSecurity extractQuotedContent "withDoubleQuotes":
// characters outside single quotes, including inside double quotes (where $ and ` expand).
func unquotedOutsideSingleQuotes(command string) string {
	var b strings.Builder
	inS, inD := false, false
	for i := 0; i < len(command); {
		c := command[i]
		if inS {
			if c == '\'' {
				inS = false
			}
			i++
			continue
		}
		if inD {
			if c == '\\' && i+1 < len(command) {
				b.WriteByte(c)
				b.WriteByte(command[i+1])
				i += 2
				continue
			}
			if c == '"' {
				inD = false
				i++
				continue
			}
			b.WriteByte(c)
			i++
			continue
		}
		if c == '\'' {
			inS = true
			i++
			continue
		}
		if c == '"' {
			inD = true
			i++
			continue
		}
		if c == '\\' && i+1 < len(command) {
			b.WriteByte(c)
			b.WriteByte(command[i+1])
			i += 2
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// hasShellQuoteSingleQuoteBug mirrors shellQuote.ts hasShellQuoteSingleQuoteBug.
func hasShellQuoteSingleQuoteBug(command string) bool {
	inS, inD := false, false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if c == '\\' && !inS {
			i++
			continue
		}
		if c == '"' && !inS {
			inD = !inD
			continue
		}
		if c == '\'' && !inD {
			inS = !inS
			if !inS {
				bs := 0
				for j := i - 1; j >= 0 && command[j] == '\\'; j-- {
					bs++
				}
				if bs > 0 && bs%2 == 1 {
					return true
				}
				if bs > 0 && bs%2 == 0 && strings.Contains(command[i+1:], "'") {
					return true
				}
			}
			continue
		}
	}
	return false
}

var bashDangerousPatterns = []struct {
	re  *regexp.Regexp
	msg string
}{
	{regexp.MustCompile(`<\(`), "process substitution <()"},
	{regexp.MustCompile(`>\(`), "process substitution >()"},
	{regexp.MustCompile(`=\(`), "Zsh process substitution =()"},
	{regexp.MustCompile(`\$\(`), "$() command substitution"},
	{regexp.MustCompile(`\$\{`), "${} parameter substitution"},
	{regexp.MustCompile(`\$\[`), "$[] legacy arithmetic expansion"},
	{regexp.MustCompile(`~\[`), "Zsh-style parameter expansion"},
	{regexp.MustCompile(`\(e:`), "Zsh-style glob qualifiers"},
	{regexp.MustCompile(`\(\+`), "Zsh glob qualifier with command execution"},
	{regexp.MustCompile(`}\s*always\s*\{`), "Zsh always block"},
	{regexp.MustCompile(`<#`), "PowerShell comment syntax"},
	{regexp.MustCompile(`(?:^|[\s;&|])=[a-zA-Z_]`), "Zsh equals expansion (=cmd)"},
}

// BashReadOnlySecurityRejectReason returns a non-empty reason if the command should not auto-approve as read-only
// (bashSecurity.ts validateDangerousPatterns + early control / shell-quote bug gates).
func BashReadOnlySecurityRejectReason(command string) string {
	if bashControlCharRE.MatchString(command) {
		return "command contains control characters that could bypass security checks"
	}
	if hasShellQuoteSingleQuoteBug(command) {
		return "command contains single-quoted backslash patterns that could confuse shell parsing"
	}
	u := unquotedOutsideSingleQuotes(command)
	if strings.Contains(u, "`") {
		return "command contains backticks for command substitution"
	}
	for _, p := range bashDangerousPatterns {
		if p.re.MatchString(u) {
			return "command contains " + p.msg
		}
	}
	return ""
}

// BashCommandSafeForAutoRun mirrors bashSecurity.ts optimistic path: false when read-only security would reject.
func BashCommandSafeForAutoRun(command string) bool {
	return BashReadOnlySecurityRejectReason(command) == ""
}
