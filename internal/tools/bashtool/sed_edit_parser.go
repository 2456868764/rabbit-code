package bashtool

import (
	"regexp"
	"strings"
)

// SedEditInfo mirrors sedEditParser.ts SedEditInfo.
type SedEditInfo struct {
	FilePath      string
	Pattern       string
	Replacement   string
	Flags         string
	ExtendedRegex bool
}

// ParseSedEditCommand mirrors sedEditParser.ts parseSedEditCommand (slash-delimited s/// only).
func ParseSedEditCommand(command string) *SedEditInfo {
	without, ok := sedWithoutPrefix(command)
	if !ok {
		return nil
	}
	tokens, err := tokenizeShellArgs(without)
	if err != nil {
		return nil
	}
	hasInPlace := false
	extended := false
	var expression, filePath string
	exprSet := false
	fileSet := false
	i := 0
	for i < len(tokens) {
		arg := tokens[i]
		if arg == "-i" || arg == "--in-place" {
			hasInPlace = true
			i++
			if i < len(tokens) {
				next := tokens[i]
				if !strings.HasPrefix(next, "-") && (next == "" || strings.HasPrefix(next, ".")) {
					i++
				}
			}
			continue
		}
		if strings.HasPrefix(arg, "-i") && arg != "-i" {
			hasInPlace = true
			i++
			continue
		}
		if arg == "-E" || arg == "-r" || arg == "--regexp-extended" {
			extended = true
			i++
			continue
		}
		if arg == "-e" || arg == "--expression" {
			if i+1 >= len(tokens) {
				return nil
			}
			if exprSet {
				return nil
			}
			expression = tokens[i+1]
			exprSet = true
			i += 2
			continue
		}
		if strings.HasPrefix(arg, "--expression=") {
			if exprSet {
				return nil
			}
			expression = strings.TrimPrefix(arg, "--expression=")
			exprSet = true
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return nil
		}
		if !exprSet {
			expression = arg
			exprSet = true
		} else if !fileSet {
			filePath = arg
			fileSet = true
		} else {
			return nil
		}
		i++
	}
	if !hasInPlace || !exprSet || !fileSet || expression == "" || filePath == "" {
		return nil
	}
	if !strings.HasPrefix(expression, "s/") {
		return nil
	}
	rest := expression[2:]
	pattern, replacement, flags, ok := parseSedSubstParts(rest)
	if !ok {
		return nil
	}
	if !sedEditValidFlagsRE.MatchString(flags) {
		return nil
	}
	return &SedEditInfo{
		FilePath:      filePath,
		Pattern:       pattern,
		Replacement:   replacement,
		Flags:         flags,
		ExtendedRegex: extended,
	}
}

func parseSedSubstParts(rest string) (pattern, replacement, flags string, ok bool) {
	state := 0 // 0=pattern 1=replacement 2=flags
	var pat, rep, fl strings.Builder
	for j := 0; j < len(rest); {
		c := rest[j]
		if c == '\\' && j+1 < len(rest) {
			switch state {
			case 0:
				pat.WriteByte(c)
				pat.WriteByte(rest[j+1])
			case 1:
				rep.WriteByte(c)
				rep.WriteByte(rest[j+1])
			default:
				fl.WriteByte(c)
				fl.WriteByte(rest[j+1])
			}
			j += 2
			continue
		}
		if c == '/' {
			switch state {
			case 0:
				state = 1
			case 1:
				state = 2
			default:
				return "", "", "", false
			}
			j++
			continue
		}
		switch state {
		case 0:
			pat.WriteByte(c)
		case 1:
			rep.WriteByte(c)
		default:
			fl.WriteByte(c)
		}
		j++
	}
	if state != 2 {
		return "", "", "", false
	}
	return pat.String(), rep.String(), fl.String(), true
}

var sedEditValidFlagsRE = regexp.MustCompile(`^[gpimIM1-9]*$`)

// Private-use runes as BRE→ERE conversion placeholders (sedEditParser.ts).
const (
	brePhBackslash = "\ue100"
	brePhPlus      = "\ue101"
	brePhQuest     = "\ue102"
	brePhPipe      = "\ue103"
	brePhLParen    = "\ue104"
	brePhRParen    = "\ue105"
)

// ApplySedSubstitution mirrors sedEditParser.ts applySedSubstitution for SedEditInfo from ParseSedEditCommand.
func ApplySedSubstitution(content string, sedInfo *SedEditInfo) string {
	if sedInfo == nil {
		return content
	}
	jsPattern := sedPatternToGoRegexpSource(sedInfo.Pattern, sedInfo.ExtendedRegex)
	var prefix strings.Builder
	if strings.ContainsAny(sedInfo.Flags, "iI") {
		prefix.WriteString("(?i)")
	}
	if strings.ContainsAny(sedInfo.Flags, "mM") {
		prefix.WriteString("(?m)")
	}
	full := prefix.String() + jsPattern
	re, err := regexp.Compile(full)
	if err != nil {
		return content
	}
	rep := strings.ReplaceAll(sedInfo.Replacement, `\/`, `/`)
	if strings.ContainsRune(sedInfo.Flags, 'g') {
		return re.ReplaceAllStringFunc(content, func(m string) string {
			return expandSedReplacement(rep, m)
		})
	}
	loc := re.FindStringIndex(content)
	if loc == nil {
		return content
	}
	match := content[loc[0]:loc[1]]
	return content[:loc[0]] + expandSedReplacement(rep, match) + content[loc[1]:]
}

func expandSedReplacement(rep string, fullMatch string) string {
	var b strings.Builder
	for i := 0; i < len(rep); {
		c := rep[i]
		if c == '\\' && i+1 < len(rep) {
			switch rep[i+1] {
			case '&':
				b.WriteByte('&')
				i += 2
				continue
			case 'n':
				b.WriteByte('\n')
				i += 2
				continue
			case '\\':
				b.WriteByte('\\')
				i += 2
				continue
			}
		}
		if c == '&' {
			b.WriteString(fullMatch)
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

func sedPatternToGoRegexpSource(pattern string, extendedRegex bool) string {
	pat := strings.ReplaceAll(pattern, `\/`, `/`)
	if extendedRegex {
		return pat
	}
	pat = strings.ReplaceAll(pat, `\\`, brePhBackslash)
	pat = strings.ReplaceAll(pat, `\+`, brePhPlus)
	pat = strings.ReplaceAll(pat, `\?`, brePhQuest)
	pat = strings.ReplaceAll(pat, `\|`, brePhPipe)
	pat = strings.ReplaceAll(pat, `\(`, brePhLParen)
	pat = strings.ReplaceAll(pat, `\)`, brePhRParen)
	pat = strings.ReplaceAll(pat, "+", `\+`)
	pat = strings.ReplaceAll(pat, "?", `\?`)
	pat = strings.ReplaceAll(pat, "|", `\|`)
	pat = strings.ReplaceAll(pat, "(", `\(`)
	pat = strings.ReplaceAll(pat, ")", `\)`)
	pat = strings.ReplaceAll(pat, brePhBackslash, `\\`)
	pat = strings.ReplaceAll(pat, brePhPlus, "+")
	pat = strings.ReplaceAll(pat, brePhQuest, "?")
	pat = strings.ReplaceAll(pat, brePhPipe, "|")
	pat = strings.ReplaceAll(pat, brePhLParen, "(")
	pat = strings.ReplaceAll(pat, brePhRParen, ")")
	return pat
}
