package bashtool

import (
	"regexp"
	"strings"
)

var (
	gitLeadRE = regexp.MustCompile(`^git(?:\s|$)`)
	cdLeadRE  = regexp.MustCompile(`^(?:cd|pushd|popd)(?:\s|$)`)
)

// IsNormalizedGitCommand mirrors bashPermissions.ts isNormalizedGitCommand (fallback path without shell-quote AST).
func IsNormalizedGitCommand(command string) bool {
	c := strings.TrimSpace(command)
	if strings.HasPrefix(c, "git ") || c == "git" {
		return true
	}
	stripped := StripSafeWrappers(c)
	if strings.HasPrefix(stripped, "git ") || stripped == "git" {
		return true
	}
	// xargs … git …
	fields := strings.Fields(stripped)
	if len(fields) >= 2 && fields[0] == "xargs" {
		for _, w := range fields[1:] {
			if w == "git" {
				return true
			}
		}
	}
	return gitLeadRE.MatchString(stripped)
}

// IsNormalizedCdCommand mirrors bashPermissions.ts isNormalizedCdCommand.
func IsNormalizedCdCommand(command string) bool {
	stripped := StripSafeWrappers(strings.TrimSpace(command))
	fields := strings.Fields(stripped)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "cd", "pushd", "popd":
		return true
	default:
		return cdLeadRE.MatchString(stripped)
	}
}

// CommandHasAnyCd mirrors bashPermissions.ts commandHasAnyCd (uses SplitCommandDeprecated ↔ splitCommand_DEPRECATED).
func CommandHasAnyCd(command string) bool {
	for _, sub := range SplitCommandDeprecated(command) {
		if IsNormalizedCdCommand(strings.TrimSpace(sub)) {
			return true
		}
	}
	return false
}
