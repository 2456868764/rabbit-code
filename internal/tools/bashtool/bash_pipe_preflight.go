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

// ValidatePipePermissionPreflight mirrors bashCommandHelpers.ts pipe segments + readOnlyValidation cd+git rule on every segment (including non-piped compounds).
func ValidatePipePermissionPreflight(command string) error {
	segs := SplitPipeSegments(command)
	if len(segs) == 0 || (len(segs) == 1 && strings.TrimSpace(segs[0]) == "") {
		segs = []string{strings.TrimSpace(command)}
	}
	if len(segs) > 1 {
		var pureCdSegs int
		var anyCd, anyGit bool
		for _, seg := range segs {
			if IsNormalizedCdCommand(strings.TrimSpace(seg)) {
				pureCdSegs++
			}
			for _, sub := range SplitCommandDeprecated(seg) {
				t := strings.TrimSpace(sub)
				if IsNormalizedCdCommand(t) {
					anyCd = true
				}
				if IsNormalizedGitCommand(t) {
					anyGit = true
				}
			}
		}
		if pureCdSegs > 1 {
			return errors.New("bashtool: multiple directory changes in one command require approval for clarity")
		}
		if anyCd && anyGit {
			return errors.New("bashtool: compound commands with cd and git require approval to prevent bare repository attacks")
		}
	}
	for _, seg := range segs {
		if err := validateSegmentCdGitCompound(seg); err != nil {
			return err
		}
	}
	return nil
}

func validateSegmentCdGitCompound(seg string) error {
	hasCd, hasGit := false, false
	for _, sub := range SplitCommandDeprecated(seg) {
		t := strings.TrimSpace(sub)
		if IsNormalizedCdCommand(t) {
			hasCd = true
		}
		if IsNormalizedGitCommand(t) {
			hasGit = true
		}
	}
	if hasCd && hasGit {
		return errors.New("bashtool: compound commands with cd and git require approval to prevent bare repository attacks")
	}
	return nil
}
