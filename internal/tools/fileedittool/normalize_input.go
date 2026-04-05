package fileedittool

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
)

var markdownPathRE = regexp.MustCompile(`\.(md|mdx)$`)

func isMarkdownPath(p string) bool {
	return markdownPathRE.MatchString(strings.ToLower(filepath.Base(p)))
}

// desanitizationOrder mirrors FileEditTool/utils.ts DESANITIZATIONS iteration order (deterministic).
var desanitizationOrder = []struct{ from, to string }{
	{"<fnr>", "<function_results>"},
	{"<n>", "<name>"},
	{"</n>", "</name>"},
	{"<o>", "<output>"},
	{"</o>", "</output>"},
	{"<e>", "<error>"},
	{"</e>", "</error>"},
	{"<s>", "<system>"},
	{"</s>", "</system>"},
	{"<r>", "<result>"},
	{"</r>", "</result>"},
	{"< META_START >", "<META_START>"},
	{"< META_END >", "<META_END>"},
	{"< EOT >", "<EOT>"},
	{"< META >", "<META>"},
	{"< SOS >", "<SOS>"},
	{"\n\nH:", "\n\nHuman:"},
	{"\n\nA:", "\n\nAssistant:"},
}

type desanitizeReplacement struct {
	from, to string
}

// DesanitizeMatchString mirrors utils.ts desanitizeMatchString.
func DesanitizeMatchString(matchString string) (result string, applied []desanitizeReplacement) {
	result = matchString
	for _, p := range desanitizationOrder {
		before := result
		result = strings.ReplaceAll(result, p.from, p.to)
		if before != result {
			applied = append(applied, desanitizeReplacement{p.from, p.to})
		}
	}
	return result, applied
}

var lineEndingRE = regexp.MustCompile(`\r\n|\r|\n`)

// StripTrailingWhitespace mirrors FileEditTool/utils.ts stripTrailingWhitespace (\s+$/ line content, preserve line breaks).
func StripTrailingWhitespace(str string) string {
	var b strings.Builder
	last := 0
	for _, loc := range lineEndingRE.FindAllStringIndex(str, -1) {
		line := str[last:loc[0]]
		b.WriteString(trimRightUnicodeSpace(line))
		b.WriteString(str[loc[0]:loc[1]])
		last = loc[1]
	}
	b.WriteString(trimRightUnicodeSpace(str[last:]))
	return b.String()
}

func trimRightUnicodeSpace(s string) string {
	// Trim trailing Unicode space (approximate JS \s on line tail; exclude \r\n already split out).
	r := []rune(s)
	i := len(r)
	for i > 0 && isLineTrailingSpace(r[i-1]) {
		i--
	}
	return string(r[:i])
}

func isLineTrailingSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\f' || r == '\v' || unicode.IsSpace(r)
}

// NormalizeSingleFileEditInput mirrors api.ts normalizeFileEditInput for a single edit (after expandPath).
func NormalizeSingleFileEditInput(userPath, abs string, in editInput, wc *filewritetool.WriteContext) editInput {
	isMarkdown := isMarkdownPath(userPath)
	norm, _, _, err := filewritetool.ReadNormalizedFileWithContext(abs, wc)
	if err != nil {
		if os.IsNotExist(err) {
			return in
		}
		return in
	}

	normalizedNew := in.NewString
	if !isMarkdown {
		normalizedNew = StripTrailingWhitespace(in.NewString)
	}

	if strings.Contains(norm, in.OldString) {
		in.NewString = normalizedNew
		return in
	}

	desOld, reps := DesanitizeMatchString(in.OldString)
	if strings.Contains(norm, desOld) {
		newStr := normalizedNew
		for _, p := range reps {
			newStr = strings.ReplaceAll(newStr, p.from, p.to)
		}
		in.OldString = desOld
		in.NewString = newStr
		return in
	}

	in.NewString = normalizedNew
	return in
}
