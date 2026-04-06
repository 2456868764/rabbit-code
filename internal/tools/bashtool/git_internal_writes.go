package bashtool

import (
	"regexp"
	"strings"
)

var reOutputRedirectTarget = regexp.MustCompile(`(?:>>|>)\s*([^\s;&|"'<>]+)`)

func isGitInternalPathToken(tok string) bool {
	tok = strings.Trim(tok, `"'`)
	tok = strings.TrimPrefix(tok, "./")
	tok = strings.TrimPrefix(tok, "/")
	if tok == "HEAD" {
		return true
	}
	if strings.HasPrefix(tok, "hooks") {
		return true
	}
	if strings.HasPrefix(tok, "refs") {
		return true
	}
	if strings.HasPrefix(tok, "objects") {
		return true
	}
	return false
}

func shellSegments(command string) []string {
	segs := SplitPipeSegments(command)
	if len(segs) == 0 || (len(segs) == 1 && strings.TrimSpace(segs[0]) == "") {
		return []string{strings.TrimSpace(command)}
	}
	return segs
}

// CommandWritesToGitInternalPaths mirrors readOnlyValidation.ts commandWritesToGitInternalPaths (subset: mkdir/touch/cp/mv + output redirects).
func CommandWritesToGitInternalPaths(command string) bool {
	for _, seg := range shellSegments(command) {
		for _, sub := range SplitCommandDeprecated(seg) {
			if subWritesGitInternal(strings.TrimSpace(sub)) {
				return true
			}
		}
	}
	return false
}

func subWritesGitInternal(sub string) bool {
	if sub == "" {
		return false
	}
	for _, m := range reOutputRedirectTarget.FindAllStringSubmatch(sub, -1) {
		if len(m) > 1 && isGitInternalPathToken(m[1]) {
			return true
		}
	}
	fields := strings.Fields(sub)
	if len(fields) == 0 {
		return false
	}
	cmd := fields[0]
	switch cmd {
	case "mkdir", "touch":
		for _, a := range fields[1:] {
			if strings.HasPrefix(a, "-") {
				continue
			}
			if isGitInternalPathToken(a) {
				return true
			}
		}
	case "cp", "mv":
		nonF := make([]string, 0, len(fields)-1)
		for _, a := range fields[1:] {
			if strings.HasPrefix(a, "-") {
				continue
			}
			nonF = append(nonF, a)
		}
		if len(nonF) == 0 {
			return false
		}
		dest := nonF[len(nonF)-1]
		if isGitInternalPathToken(dest) {
			return true
		}
	}
	return false
}
