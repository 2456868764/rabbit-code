package bashtool

import (
	"errors"
	"strings"
)

// SplitPipeSegments splits on unquoted '|' (bashCommandHelpers piped segments subset).
func SplitPipeSegments(command string) []string {
	var parts []string
	var b strings.Builder
	rs := []rune(command)
	inS, inD := false, false
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if inS {
			b.WriteRune(r)
			if r == '\'' {
				inS = false
			}
			continue
		}
		if inD {
			if r == '\\' && i+1 < len(rs) {
				b.WriteRune(r)
				b.WriteRune(rs[i+1])
				i++
				continue
			}
			b.WriteRune(r)
			if r == '"' {
				inD = false
			}
			continue
		}
		switch r {
		case '\'':
			inS = true
			b.WriteRune(r)
		case '"':
			inD = true
			b.WriteRune(r)
		case '|':
			parts = append(parts, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	parts = append(parts, strings.TrimSpace(b.String()))
	// collapse: single empty segment
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

// ValidatePipePermissionPreflight mirrors segmented checks in bashCommandHelpers.ts when pipeSegments.length > 1.
func ValidatePipePermissionPreflight(command string) error {
	segs := SplitPipeSegments(command)
	if len(segs) <= 1 {
		return nil
	}
	var pureCdSegs int
	for _, seg := range segs {
		if IsNormalizedCdCommand(strings.TrimSpace(seg)) {
			pureCdSegs++
		}
	}
	if pureCdSegs > 1 {
		return errors.New("bashtool: multiple directory changes in one command require approval for clarity")
	}
	hasCd, hasGit := false, false
	for _, seg := range segs {
		for _, sub := range SplitCommandDeprecated(seg) {
			t := strings.TrimSpace(sub)
			if IsNormalizedCdCommand(t) {
				hasCd = true
			}
			if IsNormalizedGitCommand(t) {
				hasGit = true
			}
		}
	}
	if hasCd && hasGit {
		return errors.New("bashtool: compound commands with cd and git require approval to prevent bare repository attacks")
	}
	return nil
}
