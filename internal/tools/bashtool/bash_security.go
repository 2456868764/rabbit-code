package bashtool

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/2456868764/rabbit-code/internal/readonlycmd"
)

// Mirrors bashSecurity.ts CONTROL_CHAR_RE (excludes tab, LF, CR — handled elsewhere).
var bashControlCharRE = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)

// heredocInsideSubstitutionRE mirrors bashSecurity.ts HEREDOC_IN_SUBSTITUTION.
var heredocInsideSubstitutionRE = regexp.MustCompile(`\$\([^)]*<<`)

// reFcDashE mirrors bashSecurity.ts validateZshDangerousCommands fc -e check.
var reFcDashE = regexp.MustCompile(`\s-\S*e`)

// reEnvAssignLead mirrors bashSecurity env var assignment prefix for base-command resolution.
var reEnvAssignLead = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// zshPrecmdModifiers mirrors bashSecurity.ts ZSH_PRECOMMAND_MODIFIERS.
var zshPrecmdModifiers = map[string]struct{}{
	"command": {}, "builtin": {}, "noglob": {}, "nocorrect": {},
}

// zshDangerousCommands mirrors bashSecurity.ts ZSH_DANGEROUS_COMMANDS (first word of a simple command).
var zshDangerousCommands = map[string]struct{}{
	"zmodload": {}, "emulate": {}, "sysopen": {}, "sysread": {}, "syswrite": {}, "sysseek": {},
	"zpty": {}, "ztcp": {}, "zsocket": {}, "mapfile": {}, "zf_rm": {}, "zf_mv": {}, "zf_ln": {},
	"zf_chmod": {}, "zf_chown": {}, "zf_mkdir": {}, "zf_rmdir": {}, "zf_chgrp": {},
}

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
// (bashSecurity.ts bashCommandIsSafe_DEPRECATED: control/quote bug, incomplete, validateSafeCommandSubstitution /
// isSafeHeredoc early allow, comment desync, quoted-newline, CR,
// Unicode/IFS/proc, obfuscated quotes, dangerous patterns + shell metacharacters, heredoc-in-subst, jq, dangerous vars,
// newlines, redirections, backslash ws/operators, mid-word #, zsh/fc, brace expansion, mvdan CmdSubst).
func BashReadOnlySecurityRejectReason(command string) string {
	return bashReadOnlySecurityRejectReason(command, true)
}

func bashReadOnlySecurityRejectReason(command string, allowSafeHeredocEarlyAllow bool) string {
	if bashControlCharRE.MatchString(command) {
		return "command contains control characters that could bypass security checks"
	}
	if hasShellQuoteSingleQuoteBug(command) {
		return "command contains single-quoted backslash patterns that could confuse shell parsing"
	}
	if r := bashIncompleteCommandsRejectReason(command); r != "" {
		return r
	}
	if allowSafeHeredocEarlyAllow && isSafeHeredoc(command, func(s string) string {
		return bashReadOnlySecurityRejectReason(s, false)
	}) {
		return ""
	}
	if r := bashCommentQuoteDesyncRejectReason(command); r != "" {
		return r
	}
	if r := bashQuotedNewlineRejectReason(command); r != "" {
		return r
	}
	if r := bashCarriageReturnRejectReason(command); r != "" {
		return r
	}
	if r := bashUnicodeWhitespaceRejectReason(command); r != "" {
		return r
	}
	if r := bashIFSRejectReason(command); r != "" {
		return r
	}
	if r := bashProcEnvironRejectReason(command); r != "" {
		return r
	}
	if r := bashObfuscatedQuotingRejectReason(command); r != "" {
		return r
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
	if r := bashShellMetacharactersRejectReason(u); r != "" {
		return r
	}
	if heredocInsideSubstitutionRE.MatchString(command) {
		return "command may embed a heredoc inside command substitution"
	}
	if r := bashJqCompoundRejectReason(command); r != "" {
		return r
	}
	if r := bashDangerousVarRejectReason(command); r != "" {
		return r
	}
	if r := bashNewlinesRejectReason(command); r != "" {
		return r
	}
	if r := bashRedirectionsRejectReason(command); r != "" {
		return r
	}
	if r := bashBackslashEscapedWhitespaceRejectReason(command); r != "" {
		return r
	}
	if r := bashBackslashEscapedOperatorsRejectReason(command); r != "" {
		return r
	}
	if r := bashMidWordHashRejectReason(command); r != "" {
		return r
	}
	if r := bashZshDangerousFirstWordRejectReason(command); r != "" {
		return r
	}
	if r := bashBraceExpansionRejectReason(command); r != "" {
		return r
	}
	if r := bashShellParseSecurityRejectReason(command); r != "" {
		return r
	}
	return ""
}

func bashZshDangerousFirstWordRejectReason(command string) string {
	for _, seg := range shellSegments(command) {
		for _, sub := range SplitCommandDeprecated(seg) {
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			toks, err := readonlycmd.TokenizeShellWords(sub)
			var words []string
			if err != nil || len(toks) == 0 {
				words = strings.Fields(sub)
			} else {
				words = toks
			}
			first := ""
			for _, w := range words {
				if reEnvAssignLead.MatchString(w) {
					continue
				}
				if _, mod := zshPrecmdModifiers[w]; mod {
					continue
				}
				first = w
				break
			}
			if first == "" {
				continue
			}
			if first == "fc" && reFcDashE.MatchString(sub) {
				return "command uses 'fc -e' which can execute arbitrary commands via editor"
			}
			if _, bad := zshDangerousCommands[first]; bad {
				return fmt.Sprintf("command invokes zsh-specific dangerous builtin %q", first)
			}
		}
	}
	return ""
}

// BashCommandSafeForAutoRun mirrors bashSecurity.ts optimistic path: false when read-only security would reject.
func BashCommandSafeForAutoRun(command string) bool {
	return BashReadOnlySecurityRejectReason(command) == ""
}
