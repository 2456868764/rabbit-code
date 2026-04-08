package bashtool

import (
	"regexp"
	"sort"
	"strings"
)

// reSafeHeredocOpen mirrors bashSecurity.ts isSafeHeredoc heredocPattern (quoted or backslash-escaped delimiter only).
var reSafeHeredocOpen = regexp.MustCompile(`\$\(cat[ \t]*<<(-?)[ \t]*(?:'+([A-Za-z_]\w*)'+|\\([A-Za-z_]\w*))`)

// remainingAfterSafeHeredocStripRE mirrors bashSecurity.ts isSafeHeredoc allowed remainder (ASCII space/tab only as whitespace).
var remainingAfterSafeHeredocStripRE = regexp.MustCompile(`^[a-zA-Z0-9 \t"'.\-/_@=,:+~]*$`)

// isSafeHeredoc mirrors bashSecurity.ts isSafeHeredoc: only $(cat <<'DELIM' / <<\DELIM with literal body and safe remainder + recursive security on stripped command.
func isSafeHeredoc(command string, remainderSecurity func(string) string) bool {
	if !heredocInsideSubstitutionRE.MatchString(command) {
		return false
	}

	idxs := reSafeHeredocOpen.FindAllStringSubmatchIndex(command, -1)
	type heredocMatch struct {
		start, operatorEnd int
		delimiter          string
		isDash             bool
	}
	var matches []heredocMatch
	for _, ix := range idxs {
		if len(ix) < 8 {
			continue
		}
		dash := ""
		if ix[2] >= 0 && ix[3] >= 0 {
			dash = command[ix[2]:ix[3]]
		}
		var delim string
		if ix[4] >= 0 && ix[5] >= 0 {
			delim = command[ix[4]:ix[5]]
		} else if ix[6] >= 0 && ix[7] >= 0 {
			delim = command[ix[6]:ix[7]]
		}
		if delim == "" {
			continue
		}
		matches = append(matches, heredocMatch{
			start:       ix[0],
			operatorEnd: ix[1],
			delimiter:   delim,
			isDash:      dash == "-",
		})
	}
	if len(matches) == 0 {
		return false
	}

	type verifiedRange struct{ start, end int }
	var verifiedRanges []verifiedRange

	for _, m := range matches {
		afterOperator := command[m.operatorEnd:]
		openLineEnd := strings.IndexByte(afterOperator, '\n')
		if openLineEnd == -1 {
			return false
		}
		openLineTail := afterOperator[:openLineEnd]
		if !isOnlyHorizontalSpace(openLineTail) {
			return false
		}
		bodyStart := m.operatorEnd + openLineEnd + 1
		body := command[bodyStart:]
		bodyLines := strings.Split(body, "\n")

		closingLineIdx := -1
		closeParenLineIdx := -1
		closeParenColIdx := -1

		for i := 0; i < len(bodyLines); i++ {
			rawLine := bodyLines[i]
			line := rawLine
			if m.isDash {
				line = strings.TrimLeft(rawLine, "\t")
			}

			if line == m.delimiter {
				closingLineIdx = i
				if i+1 >= len(bodyLines) {
					return false
				}
				nextLine := bodyLines[i+1]
				parenIdx := leadingSpaceThenParen(nextLine)
				if parenIdx < 0 {
					return false
				}
				closeParenLineIdx = i + 1
				closeParenColIdx = parenIdx
				break
			}

			if strings.HasPrefix(line, m.delimiter) {
				afterDelim := line[len(m.delimiter):]
				parenIdx := leadingSpaceThenParen(afterDelim)
				if parenIdx >= 0 {
					closingLineIdx = i
					closeParenLineIdx = i
					tabPrefix := ""
					if m.isDash {
						tabPrefix = leadingTabs(rawLine)
					}
					closeParenColIdx = len(tabPrefix) + len(m.delimiter) + parenIdx
					break
				}
				if len(afterDelim) > 0 {
					c0 := afterDelim[0]
					if c0 == ')' || c0 == '}' || c0 == '`' || c0 == '|' || c0 == '&' || c0 == ';' ||
						c0 == '(' || c0 == '<' || c0 == '>' {
						return false
					}
				}
			}
		}

		if closingLineIdx == -1 {
			return false
		}

		endPos := bodyStart
		for i := 0; i < closeParenLineIdx; i++ {
			endPos += len(bodyLines[i]) + 1
		}
		endPos += closeParenColIdx + 1

		verifiedRanges = append(verifiedRanges, verifiedRange{start: m.start, end: endPos})
	}

	for i := range verifiedRanges {
		for j := range verifiedRanges {
			if i == j {
				continue
			}
			oi, oj := verifiedRanges[i], verifiedRanges[j]
			if oj.start > oi.start && oj.start < oi.end {
				return false
			}
		}
	}

	remaining := command
	sorted := append([]verifiedRange(nil), verifiedRanges...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].start > sorted[j].start })
	for _, r := range sorted {
		remaining = remaining[:r.start] + remaining[r.end:]
	}

	if strings.TrimSpace(remaining) != "" {
		firstStart := verifiedRanges[0].start
		for _, r := range verifiedRanges[1:] {
			if r.start < firstStart {
				firstStart = r.start
			}
		}
		prefix := command[:firstStart]
		if strings.TrimSpace(prefix) == "" {
			return false
		}
	}

	if !remainingAfterSafeHeredocStripRE.MatchString(remaining) {
		return false
	}

	if remainderSecurity(remaining) != "" {
		return false
	}
	return true
}

func isOnlyHorizontalSpace(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return true
}

// leadingSpaceThenParen returns column index of ')' if s matches ^[ \t]*\), else -1.
func leadingSpaceThenParen(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i < len(s) && s[i] == ')' {
		return i
	}
	return -1
}

func leadingTabs(s string) string {
	i := 0
	for i < len(s) && s[i] == '\t' {
		i++
	}
	return s[:i]
}
