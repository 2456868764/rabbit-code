package bashtool

import (
	"regexp"
	"strings"
)

type destructivePattern struct {
	re      *regexp.Regexp
	warning string
}

// destructivePatterns mirrors destructiveCommandWarning.ts DESTRUCTIVE_PATTERNS.
// git clean uses gitCleanDestructive (Go regexp has no negative lookahead).
var destructivePatterns = []destructivePattern{
	{regexp.MustCompile(`\bgit\s+reset\s+--hard\b`), "Note: may discard uncommitted changes"},
	{regexp.MustCompile(`\bgit\s+push\b[^;&|\n]*[ \t](--force|--force-with-lease|-f)\b`), "Note: may overwrite remote history"},
	{regexp.MustCompile(`\bgit\s+checkout\s+(--\s+)?\.[ \t]*($|[;&|\n])`), "Note: may discard all working tree changes"},
	{regexp.MustCompile(`\bgit\s+restore\s+(--\s+)?\.[ \t]*($|[;&|\n])`), "Note: may discard all working tree changes"},
	{regexp.MustCompile(`\bgit\s+stash[ \t]+(drop|clear)\b`), "Note: may permanently remove stashed changes"},
	{regexp.MustCompile(`\bgit\s+branch\s+(-D[ \t]|--delete\s+--force|--force\s+--delete)\b`), "Note: may force-delete a branch"},
	{regexp.MustCompile(`\bgit\s+(commit|push|merge)\b[^;&|\n]*--no-verify\b`), "Note: may skip safety hooks"},
	{regexp.MustCompile(`\bgit\s+commit\b[^;&|\n]*--amend\b`), "Note: may rewrite the last commit"},
	{regexp.MustCompile(`(^|[;&|\n]\s*)rm\s+-[a-zA-Z]*[rR][a-zA-Z]*f|(^|[;&|\n]\s*)rm\s+-[a-zA-Z]*f[a-zA-Z]*[rR]`), "Note: may recursively force-remove files"},
	{regexp.MustCompile(`(^|[;&|\n]\s*)rm\s+-[a-zA-Z]*[rR]`), "Note: may recursively remove files"},
	{regexp.MustCompile(`(^|[;&|\n]\s*)rm\s+-[a-zA-Z]*f`), "Note: may force-remove files"},
	{regexp.MustCompile(`(?i)\b(DROP|TRUNCATE)\s+(TABLE|DATABASE|SCHEMA)\b`), "Note: may drop or truncate database objects"},
	{regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+\w+[ \t]*(;|"|'|\n|$)`), "Note: may delete all rows from a database table"},
	{regexp.MustCompile(`\bkubectl\s+delete\b`), "Note: may delete Kubernetes resources"},
	{regexp.MustCompile(`\bterraform\s+destroy\b`), "Note: may destroy Terraform infrastructure"},
}

var (
	reGitCleanWord = regexp.MustCompile(`\bgit\s+clean\b`)
	// Same intent as TS: short-option cluster ending in "n" (e.g. -n, -dn) flags dry-run style.
	reShortFlagsEndingN = regexp.MustCompile(`-[a-zA-Z]*n(?:[^a-zA-Z]|$)`)
	reShortFlagsWithF   = regexp.MustCompile(`-[a-zA-Z]*f`)
)

// gitCleanDestructive matches TS git clean line: force (-f in a short-flag cluster) but not dry-run (-n cluster or --dry-run) in the same shell segment.
func gitCleanDestructive(command string) bool {
	loc := 0
	for loc < len(command) {
		rel := reGitCleanWord.FindStringIndex(command[loc:])
		if rel == nil {
			return false
		}
		after := loc + rel[1]
		end := len(command)
		for i := after; i < len(command); i++ {
			switch command[i] {
			case ';', '&', '|', '\n':
				end = i
			}
			if end < len(command) {
				break
			}
		}
		seg := command[after:end]
		if strings.Contains(seg, "--dry-run") || reShortFlagsEndingN.MatchString(seg) || !reShortFlagsWithF.MatchString(seg) {
			loc = loc + rel[0] + 1
			continue
		}
		return true
	}
	return false
}

// GetDestructiveCommandWarning mirrors destructiveCommandWarning.ts getDestructiveCommandWarning.
func GetDestructiveCommandWarning(command string) string {
	if gitCleanDestructive(command) {
		return "Note: may permanently delete untracked files"
	}
	for _, p := range destructivePatterns {
		if p.re.MatchString(command) {
			return p.warning
		}
	}
	return ""
}
