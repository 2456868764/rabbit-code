package fileedittool

import (
	"strings"
	"unicode"
)

// Curly quotes mirror FileEditTool/utils.ts constants.
const (
	leftSingle  = '\u2018'
	rightSingle = '\u2019'
	leftDouble  = '\u201c'
	rightDouble = '\u201d'
)

// NormalizeQuotes mirrors utils.ts normalizeQuotes.
func NormalizeQuotes(str string) string {
	s := str
	s = strings.ReplaceAll(s, string(leftSingle), "'")
	s = strings.ReplaceAll(s, string(rightSingle), "'")
	s = strings.ReplaceAll(s, string(leftDouble), `"`)
	s = strings.ReplaceAll(s, string(rightDouble), `"`)
	return s
}

// FindActualString mirrors utils.ts findActualString (rune-aligned after quote normalization).
func FindActualString(fileContent, searchString string) string {
	if strings.Contains(fileContent, searchString) {
		return searchString
	}
	nSearch := NormalizeQuotes(searchString)
	nFile := NormalizeQuotes(fileContent)
	rs := []rune(nSearch)
	rf := []rune(nFile)
	orig := []rune(fileContent)
	if len(rs) == 0 {
		return ""
	}
	for i := 0; i+len(rs) <= len(rf); i++ {
		ok := true
		for j := range rs {
			if rf[i+j] != rs[j] {
				ok = false
				break
			}
		}
		if ok {
			if i+len(rs) > len(orig) {
				return ""
			}
			return string(orig[i : i+len(rs)])
		}
	}
	return ""
}

func isOpeningContext(chars []rune, index int) bool {
	if index == 0 {
		return true
	}
	prev := chars[index-1]
	switch prev {
	case ' ', '\t', '\n', '\r', '(', '[', '{', '\u2014', '\u2013':
		return true
	default:
		return false
	}
}

func applyCurlyDoubleQuotes(str string) string {
	chars := []rune(str)
	var b strings.Builder
	for i, ch := range chars {
		if ch == '"' {
			if isOpeningContext(chars, i) {
				b.WriteRune(leftDouble)
			} else {
				b.WriteRune(rightDouble)
			}
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func applyCurlySingleQuotes(str string) string {
	chars := []rune(str)
	var b strings.Builder
	for i, ch := range chars {
		if ch == '\'' {
			var prev, next rune
			if i > 0 {
				prev = chars[i-1]
			}
			if i < len(chars)-1 {
				next = chars[i+1]
			}
			prevLetter := prev != 0 && unicode.IsLetter(prev)
			nextLetter := next != 0 && unicode.IsLetter(next)
			if prevLetter && nextLetter {
				b.WriteRune(rightSingle)
			} else if isOpeningContext(chars, i) {
				b.WriteRune(leftSingle)
			} else {
				b.WriteRune(rightSingle)
			}
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// PreserveQuoteStyle mirrors utils.ts preserveQuoteStyle.
func PreserveQuoteStyle(oldString, actualOldString, newString string) string {
	if oldString == actualOldString {
		return newString
	}
	hasDouble := strings.ContainsRune(actualOldString, leftDouble) ||
		strings.ContainsRune(actualOldString, rightDouble)
	hasSingle := strings.ContainsRune(actualOldString, leftSingle) ||
		strings.ContainsRune(actualOldString, rightSingle)
	out := newString
	if hasDouble {
		out = applyCurlyDoubleQuotes(out)
	}
	if hasSingle {
		out = applyCurlySingleQuotes(out)
	}
	return out
}

// ApplyEditToFile mirrors utils.ts applyEditToFile.
func ApplyEditToFile(originalContent, oldString, newString string, replaceAll bool) string {
	if newString != "" {
		if replaceAll {
			return strings.ReplaceAll(originalContent, oldString, newString)
		}
		return strings.Replace(originalContent, oldString, newString, 1)
	}
	stripTrailingNewline := !strings.HasSuffix(oldString, "\n") &&
		strings.Contains(originalContent, oldString+"\n")
	if stripTrailingNewline {
		if replaceAll {
			return strings.ReplaceAll(originalContent, oldString+"\n", newString)
		}
		return strings.Replace(originalContent, oldString+"\n", newString, 1)
	}
	if replaceAll {
		return strings.ReplaceAll(originalContent, oldString, newString)
	}
	return strings.Replace(originalContent, oldString, newString, 1)
}
