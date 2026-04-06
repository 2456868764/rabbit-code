package bashtool

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

var sedLeadRE = regexp.MustCompile(`(?i)^\s*sed\s+`)

// tokenizeShellArgs splits sed arguments after the command name (single/double quotes, no command substitution).
func tokenizeShellArgs(s string) ([]string, error) {
	var toks []string
	var b strings.Builder
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
			b.Reset()
			for i < len(s) && s[i] != '\'' {
				b.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				return nil, errors.New("unclosed quote")
			}
			toks = append(toks, b.String())
			i++
			continue
		}
		if s[i] == '"' {
			i++
			b.Reset()
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					b.WriteByte(s[i+1])
					i += 2
					continue
				}
				if s[i] == '"' {
					break
				}
				b.WriteByte(s[i])
				i++
			}
			if i >= len(s) {
				return nil, errors.New("unclosed quote")
			}
			toks = append(toks, b.String())
			i++
			continue
		}
		b.Reset()
		for i < len(s) && s[i] != ' ' && s[i] != '\t' {
			b.WriteByte(s[i])
			i++
		}
		toks = append(toks, b.String())
	}
	return toks, nil
}

func sedWithoutPrefix(command string) (string, bool) {
	command = strings.TrimSpace(command)
	loc := sedLeadRE.FindStringIndex(command)
	if loc == nil {
		return "", false
	}
	return command[loc[1]:], true
}

func collectSedFlags(tokens []string) []string {
	var flags []string
	for _, arg := range tokens {
		if strings.HasPrefix(arg, "-") && arg != "--" {
			flags = append(flags, arg)
		}
	}
	return flags
}

func validateSedFlagsAgainstAllowlist(flags []string, allowed []string) bool {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = struct{}{}
	}
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--") {
			if _, ok := allowedSet[flag]; !ok {
				return false
			}
			continue
		}
		if len(flag) > 2 && strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") {
			for j := 1; j < len(flag); j++ {
				sf := "-" + string(flag[j])
				if _, ok := allowedSet[sf]; !ok {
					return false
				}
			}
			continue
		}
		if _, ok := allowedSet[flag]; !ok {
			return false
		}
	}
	return true
}

// IsPrintCommand mirrors sedValidation.ts isPrintCommand.
func IsPrintCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}
	return regexp.MustCompile(`^(?:\d+|\d+,\d+)?p$`).MatchString(cmd)
}

func isLinePrintingSed(command string, expressions []string) bool {
	without, ok := sedWithoutPrefix(command)
	if !ok {
		return false
	}
	tokens, err := tokenizeShellArgs(without)
	if err != nil {
		return false
	}
	flags := collectSedFlags(tokens)
	allowed := []string{
		"-n", "--quiet", "--silent", "-E", "--regexp-extended", "-r",
		"-z", "--zero-terminated", "--posix",
	}
	if !validateSedFlagsAgainstAllowlist(flags, allowed) {
		return false
	}
	hasN := false
	for _, f := range flags {
		if f == "-n" || f == "--quiet" || f == "--silent" {
			hasN = true
			break
		}
		if strings.HasPrefix(f, "-") && !strings.HasPrefix(f, "--") && strings.ContainsRune(f, 'n') {
			hasN = true
			break
		}
	}
	if !hasN {
		return false
	}
	if len(expressions) == 0 {
		return false
	}
	for _, expr := range expressions {
		for _, part := range strings.Split(expr, ";") {
			if !IsPrintCommand(strings.TrimSpace(part)) {
				return false
			}
		}
	}
	return true
}

func isSubstitutionSed(command string, expressions []string, hasFileArguments, allowFileWrites bool) bool {
	if !allowFileWrites && hasFileArguments {
		return false
	}
	without, ok := sedWithoutPrefix(command)
	if !ok {
		return false
	}
	tokens, err := tokenizeShellArgs(without)
	if err != nil {
		return false
	}
	flags := collectSedFlags(tokens)
	allowed := []string{"-E", "--regexp-extended", "-r", "--posix"}
	if allowFileWrites {
		allowed = append(allowed, "-i", "--in-place")
	}
	if !validateSedFlagsAgainstAllowlist(flags, allowed) {
		return false
	}
	if len(expressions) != 1 {
		return false
	}
	expr := strings.TrimSpace(expressions[0])
	if !strings.HasPrefix(expr, "s") {
		return false
	}
	if !strings.HasPrefix(expr, "s/") {
		return false
	}
	rest := expr[2:]
	delims := 0
	lastDelim := -1
	for i := 0; i < len(rest); {
		if rest[i] == '\\' {
			i += 2
			if i > len(rest) {
				break
			}
			continue
		}
		if rest[i] == '/' {
			delims++
			lastDelim = i
			i++
			continue
		}
		i++
	}
	if delims != 2 || lastDelim < 0 {
		return false
	}
	exprFlags := rest[lastDelim+1:]
	if !regexp.MustCompile(`^[gpimIM]*[1-9]?[gpimIM]*$`).MatchString(exprFlags) {
		return false
	}
	return true
}

var (
	reNonASCII          = regexp.MustCompile(`[^\x01-\x7F]`)
	reAddrTilde         = regexp.MustCompile(`\d\s*~\s*\d|,\s*~\s*\d|\$\s*~\s*\d`)
	reCommaOffset       = regexp.MustCompile(`,\s*[+-]`)
	reSedBackslashDelim = regexp.MustCompile(`s\\|\\[|#%@]`)
	reSlashWW           = regexp.MustCompile(`\\/.*[wW]`)
	reSlashSpaceWE      = regexp.MustCompile(`\/[^/]*\s+[wWeE]`)
	reMalformedSubst    = regexp.MustCompile(`^s\/[^/]*\/[^/]*\/[^/]*$`)
	reWfile             = regexp.MustCompile(`^[wW]\s*\S+`)
	reLineW             = regexp.MustCompile(`^\d+\s*[wW]\s*\S+`)
	reDollarW           = regexp.MustCompile(`^\$\s*[wW]\s*\S+`)
	rePatternW          = regexp.MustCompile(`^/[^/]*/[IMim]*\s*[wW]\s*\S+`)
	reRangeW            = regexp.MustCompile(`^\d+,\d+\s*[wW]\s*\S+`)
	reRangeDollarW      = regexp.MustCompile(`^\d+,\$\s*[wW]\s*\S+`)
	rePatRangeW         = regexp.MustCompile(`^/[^/]*/[IMim]*,/[^/]*/[IMim]*\s*[wW]\s*\S+`)
	reEcmd              = regexp.MustCompile(`^e|^\d+\s*e|^\$\s*e|^/[^/]*/[IMim]*\s*e|^\d+,\d+\s*e|^\d+,\$\s*e|^/[^/]*/[IMim]*,/[^/]*/[IMim]*\s*e`)
	reYwithWE           = regexp.MustCompile(`y([^\\\n])`)
)

// isProperSedSubstitutionForm mirrors sedValidation.ts "properSubst" (three delimited segments after s).
func isProperSedSubstitutionForm(cmd string) bool {
	if len(cmd) < 4 || cmd[0] != 's' {
		return false
	}
	delim, w := utf8.DecodeRuneInString(cmd[1:])
	if delim == utf8.RuneError || w == 0 || delim == '\n' || delim == '\\' {
		return false
	}
	state := 0
	i := 1 + w
	for i < len(cmd) {
		r, rw := utf8.DecodeRuneInString(cmd[i:])
		if r == utf8.RuneError {
			break
		}
		if r == '\\' {
			i += rw
			if i < len(cmd) {
				_, w2 := utf8.DecodeRuneInString(cmd[i:])
				i += w2
			}
			continue
		}
		if r == delim {
			state++
			if state > 2 {
				return false
			}
			i += rw
			continue
		}
		i += rw
	}
	return state == 2
}

func sedSubstitutionDangerousFlags(expression string) bool {
	if len(expression) < 4 || expression[0] != 's' {
		return false
	}
	delim, w := utf8.DecodeRuneInString(expression[1:])
	if delim == utf8.RuneError || w == 0 {
		return false
	}
	state := 0
	lastDelimEnd := 0
	i := 1 + w
	for i < len(expression) {
		r, rw := utf8.DecodeRuneInString(expression[i:])
		if r == '\\' {
			i += rw
			if i < len(expression) {
				_, w2 := utf8.DecodeRuneInString(expression[i:])
				i += w2
			}
			continue
		}
		if r == delim {
			state++
			i += rw
			lastDelimEnd = i
			continue
		}
		i += rw
	}
	if state != 2 {
		return false
	}
	flags := expression[lastDelimEnd:]
	return strings.ContainsAny(flags, "wW") || strings.ContainsAny(flags, "eE")
}

func containsDangerousSedOperations(expression string) bool {
	cmd := strings.TrimSpace(expression)
	if cmd == "" {
		return false
	}
	if reNonASCII.MatchString(cmd) {
		return true
	}
	if strings.ContainsAny(cmd, "{}") {
		return true
	}
	if strings.ContainsRune(cmd, '\n') {
		return true
	}
	if idx := strings.IndexRune(cmd, '#'); idx >= 0 {
		if !(idx > 0 && cmd[idx-1] == 's') {
			return true
		}
	}
	if strings.HasPrefix(cmd, "!") || regexp.MustCompile(`[/\d$]!`).MatchString(cmd) {
		return true
	}
	if reAddrTilde.MatchString(cmd) {
		return true
	}
	if strings.HasPrefix(cmd, ",") || reCommaOffset.MatchString(cmd) {
		return true
	}
	if reSedBackslashDelim.MatchString(cmd) {
		return true
	}
	if reSlashWW.MatchString(cmd) {
		return true
	}
	if reSlashSpaceWE.MatchString(cmd) {
		return true
	}
	if strings.HasPrefix(cmd, "s/") && !reMalformedSubst.MatchString(cmd) {
		return true
	}
	if len(cmd) >= 2 && cmd[0] == 's' && regexp.MustCompile(`[wWeE]$`).MatchString(cmd) {
		if !isProperSedSubstitutionForm(cmd) {
			return true
		}
	}
	if reWfile.MatchString(cmd) || reLineW.MatchString(cmd) || reDollarW.MatchString(cmd) ||
		rePatternW.MatchString(cmd) || reRangeW.MatchString(cmd) || reRangeDollarW.MatchString(cmd) ||
		rePatRangeW.MatchString(cmd) {
		return true
	}
	if reEcmd.MatchString(cmd) {
		return true
	}
	if sedSubstitutionDangerousFlags(cmd) {
		return true
	}
	if reYwithWE.MatchString(cmd) && regexp.MustCompile(`[wWeE]`).MatchString(cmd) {
		return true
	}
	return false
}

// ExtractSedExpressions mirrors sedValidation.ts extractSedExpressions.
func ExtractSedExpressions(command string) ([]string, error) {
	without, ok := sedWithoutPrefix(command)
	if !ok {
		return nil, nil
	}
	if matched, _ := regexp.MatchString(`-e[wWe]|-w[eE]`, without); matched {
		return nil, errors.New("dangerous sed flag combination")
	}
	tokens, err := tokenizeShellArgs(without)
	if err != nil {
		return nil, err
	}
	var expressions []string
	foundEFlag := false
	foundExpr := false
	for i := 0; i < len(tokens); i++ {
		arg := tokens[i]
		if (arg == "-e" || arg == "--expression") && i+1 < len(tokens) {
			foundEFlag = true
			expressions = append(expressions, tokens[i+1])
			i++
			continue
		}
		if strings.HasPrefix(arg, "--expression=") {
			foundEFlag = true
			expressions = append(expressions, strings.TrimPrefix(arg, "--expression="))
			continue
		}
		if strings.HasPrefix(arg, "-e=") {
			foundEFlag = true
			expressions = append(expressions, strings.TrimPrefix(arg, "-e="))
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if !foundEFlag && !foundExpr {
			expressions = append(expressions, arg)
			foundExpr = true
			continue
		}
		break
	}
	return expressions, nil
}

// HasSedFileArgs mirrors sedValidation.ts hasFileArgs.
func HasSedFileArgs(command string) bool {
	without, ok := sedWithoutPrefix(command)
	if !ok {
		return false
	}
	tokens, err := tokenizeShellArgs(without)
	if err != nil {
		return true
	}
	argCount := 0
	hasEFlag := false
	for i := 0; i < len(tokens); i++ {
		arg := tokens[i]
		if (arg == "-e" || arg == "--expression") && i+1 < len(tokens) {
			hasEFlag = true
			i++
			continue
		}
		if strings.HasPrefix(arg, "--expression=") || strings.HasPrefix(arg, "-e=") {
			hasEFlag = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		argCount++
		if hasEFlag {
			return true
		}
		if argCount > 1 {
			return true
		}
	}
	return false
}

// SedCommandAllowedByAllowlist mirrors sedValidation.ts sedCommandIsAllowedByAllowlist (allowFileWrites=false).
func SedCommandAllowedByAllowlist(command string) bool {
	return sedCommandAllowedByAllowlist(command, false)
}

// SedCommandAllowedByAllowlistWithFileWrites mirrors sedValidation.ts with allowFileWrites=true (acceptEdits-style).
func SedCommandAllowedByAllowlistWithFileWrites(command string) bool {
	return sedCommandAllowedByAllowlist(command, true)
}

func sedCommandAllowedByAllowlist(command string, allowFileWrites bool) bool {
	expressions, err := ExtractSedExpressions(command)
	if err != nil || len(expressions) == 0 {
		return false
	}
	hasFiles := HasSedFileArgs(command)
	var p1, p2 bool
	if allowFileWrites {
		p2 = isSubstitutionSed(command, expressions, hasFiles, true)
	} else {
		p1 = isLinePrintingSed(command, expressions)
		p2 = isSubstitutionSed(command, expressions, hasFiles, false)
	}
	if !p1 && !p2 {
		return false
	}
	for _, expr := range expressions {
		if p2 && strings.ContainsRune(expr, ';') {
			return false
		}
	}
	for _, expr := range expressions {
		if containsDangerousSedOperations(expr) {
			return false
		}
	}
	return true
}
