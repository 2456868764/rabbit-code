package bashtool

import (
	"fmt"
	"regexp"
	"strings"
)

// shell-quote parse.js parity (npm shell-quote) for validateMalformedTokenInjection / tryParseShellCommand.

var sqControlSrc = `(?:` + strings.Join([]string{
	`\|\|`,
	`\&\&`,
	`;;`,
	`\|\&`,
	`\<\(`,
	`\<\<\<`,
	`>>`,
	`>\&`,
	`<\&`,
	`[&;()|<>]`,
}, "|") + `)`

var sqControlRE = regexp.MustCompile("^" + sqControlSrc + "$")

var sqChunkerRE *regexp.Regexp

func init() {
	dq := "\""
	tab := "\t"
	sq := "'"
	sqChunkerRE = regexp.MustCompile(
		`((?:\|\||\&\&|;;|\|\&|\<\(|\<\<\<|>>|>\&|<\&|[&;()|<>]))` +
			`|((\\[` + sq + dq + `|&;()<> ` + tab + `]|[^\s` + sq + dq + `|&;()<> ` + tab + `])+|` +
			dq + `((\\` + dq + `|[^` + dq + `])*?)` + dq + `|` + sq + `((\\` + sq + `|[^` + sq + `])*?)` + sq + `)+`,
	)
}

type sqParsedEntry interface {
	sqEntry()
}

type sqOp struct{ op string }

func (sqOp) sqEntry() {}

type sqStr struct{ s string }

func (sqStr) sqEntry() {}

type sqGlob struct{ pattern string }

func (sqGlob) sqEntry() {}

type sqComment struct{ text string }

func (sqComment) sqEntry() {}

func sqGetVar(env map[string]string, pre, key string) string {
	if env == nil {
		env = map[string]string{}
	}
	r, ok := env[key]
	if !ok && key != "" {
		return ""
	}
	if !ok {
		return "$"
	}
	return pre + r
}

// sqParseEnvVar parses from index of '$'; returns index of first byte after expansion and expanded value.
func sqParseEnvVar(s string, dollar int, env map[string]string) (int, string, error) {
	if dollar >= len(s) || s[dollar] != '$' {
		return dollar, "", fmt.Errorf("expected $")
	}
	i := dollar + 1
	if i >= len(s) {
		return len(s), "$", nil
	}
	switch s[i] {
	case '{':
		i++
		if i < len(s) && s[i] == '}' {
			return 0, "", fmt.Errorf("bad substitution")
		}
		endRel := strings.IndexByte(s[i:], '}')
		if endRel < 0 {
			return 0, "", fmt.Errorf("bad substitution")
		}
		name := s[i : i+endRel]
		next := i + endRel + 1
		return next, sqGetVar(env, "", name), nil
	default:
		ch := s[i]
		if strings.ContainsRune("*@#?$!-_", rune(ch)) {
			return i + 1, sqGetVar(env, "", string(ch)), nil
		}
		j := i
		for j < len(s) {
			c := s[j]
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				break
			}
			j++
		}
		if j == i {
			return i, "$", nil
		}
		return j, sqGetVar(env, "", s[i:j]), nil
	}
}

func sqProcessChunk(full string, absStart int, chunk string, env map[string]string, commented *bool) (any, error) {
	if chunk == "" || *commented {
		return nil, nil
	}
	if sqControlRE.MatchString(chunk) {
		return sqOp{op: chunk}, nil
	}

	const BS = '\\'
	const SQ = '\''
	const DQ = '"'
	const DS = '$'

	var out strings.Builder
	var quote rune
	esc := false
	isGlob := false

	for i := 0; i < len(chunk); {
		c := chunk[i]
		if !esc && quote == 0 && (c == '*' || c == '?') {
			isGlob = true
		}
		if esc {
			out.WriteByte(c)
			esc = false
			i++
			continue
		}
		if quote != 0 {
			if rune(c) == quote {
				quote = 0
				i++
				continue
			}
			if quote == SQ {
				out.WriteByte(c)
				i++
				continue
			}
			// double-quoted
			if c == BS {
				i++
				if i >= len(chunk) {
					out.WriteByte(BS)
					break
				}
				c2 := chunk[i]
				if c2 == DQ || c2 == BS || c2 == DS {
					out.WriteByte(c2)
				} else {
					out.WriteByte(BS)
					out.WriteByte(c2)
				}
				i++
				continue
			}
			if c == DS {
				next, v, err := sqParseEnvVar(chunk, i, env)
				if err != nil {
					return nil, err
				}
				out.WriteString(v)
				i = next
				continue
			}
			out.WriteByte(c)
			i++
			continue
		}
		if c == DQ || c == SQ {
			quote = rune(c)
			i++
			continue
		}
		if sqControlRE.MatchString(string(c)) {
			return sqOp{op: chunk}, nil
		}
		if c == '#' {
			*commented = true
			rest := full[absStart+i+1:]
			co := sqComment{text: rest}
			if out.Len() > 0 {
				return []sqParsedEntry{sqStr{s: out.String()}, co}, nil
			}
			return []sqParsedEntry{co}, nil
		}
		if c == BS {
			esc = true
			i++
			continue
		}
		if c == DS {
			next, v, err := sqParseEnvVar(chunk, i, env)
			if err != nil {
				return nil, err
			}
			out.WriteString(v)
			i = next
			continue
		}
		out.WriteByte(c)
		i++
	}

	if isGlob {
		return sqGlob{pattern: out.String()}, nil
	}
	return sqStr{s: out.String()}, nil
}

func sqFlattenReduce(parts []any) []sqParsedEntry {
	var acc []sqParsedEntry
	for _, arg := range parts {
		if arg == nil {
			continue
		}
		switch v := arg.(type) {
		case []sqParsedEntry:
			acc = append(acc, v...)
		default:
			if e, ok := arg.(sqParsedEntry); ok {
				acc = append(acc, e)
			}
		}
	}
	return acc
}

// tryParseShellQuote mirrors shell-quote parse(cmd) with empty env {} (tryParseShellCommand default).
func tryParseShellQuote(cmd string) ([]sqParsedEntry, error) {
	env := map[string]string{}
	if cmd == "" {
		return nil, nil
	}
	locs := sqChunkerRE.FindAllStringSubmatchIndex(cmd, -1)
	if len(locs) == 0 {
		return nil, nil
	}
	commented := false
	var mapped []any
	for _, loc := range locs {
		if len(loc) < 2 {
			continue
		}
		chunk := cmd[loc[0]:loc[1]]
		absStart := loc[0]
		res, err := sqProcessChunk(cmd, absStart, chunk, env, &commented)
		if err != nil {
			return nil, err
		}
		mapped = append(mapped, res)
	}
	return sqFlattenReduce(mapped), nil
}

func shellQuoteHasCommandSeparator(entries []sqParsedEntry) bool {
	for _, e := range entries {
		o, ok := e.(sqOp)
		if !ok {
			continue
		}
		switch o.op {
		case ";", "&&", "||":
			return true
		}
	}
	return false
}

func shellQuoteHasMalformedTokens(command string, entries []sqParsedEntry) bool {
	inSingle, inDouble := false, false
	doubleCount, singleCount := 0, 0
	for i := 0; i < len(command); i++ {
		c := command[i]
		if c == '\\' && !inSingle {
			i++
			continue
		}
		if c == '"' && !inSingle {
			doubleCount++
			inDouble = !inDouble
		} else if c == '\'' && !inDouble {
			singleCount++
			inSingle = !inSingle
		}
	}
	if doubleCount%2 != 0 || singleCount%2 != 0 {
		return true
	}

	for _, e := range entries {
		s, ok := e.(sqStr)
		if !ok {
			continue
		}
		entry := s.s
		if strings.Count(entry, "{") != strings.Count(entry, "}") {
			return true
		}
		if strings.Count(entry, "(") != strings.Count(entry, ")") {
			return true
		}
		if strings.Count(entry, "[") != strings.Count(entry, "]") {
			return true
		}
		if countUnescapedChar(entry, '"')%2 != 0 {
			return true
		}
		if countUnescapedChar(entry, '\'')%2 != 0 {
			return true
		}
	}
	return false
}

func countUnescapedChar(entry string, q byte) int {
	n := 0
	for i := 0; i < len(entry); i++ {
		if entry[i] == '\\' {
			i++
			continue
		}
		if entry[i] == q {
			n++
		}
	}
	return n
}

func bashMalformedTokenInjectionRejectReason(command string) string {
	entries, err := tryParseShellQuote(command)
	if err != nil {
		return ""
	}
	if !shellQuoteHasCommandSeparator(entries) {
		return ""
	}
	if shellQuoteHasMalformedTokens(command, entries) {
		return "command contains ambiguous syntax with command separators that could be misinterpreted"
	}
	return ""
}
