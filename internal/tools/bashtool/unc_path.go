package bashtool

import (
	"regexp"
	"runtime"
	"strings"
)

var (
	reUNCBackslash = regexp.MustCompile(`(?i)\\\\[^\s\\/]+(?:@(?:\d+|ssl))?(?:[\\/]|$|\s)`)
	reUNCForward   = regexp.MustCompile(`(?i)(?:^|[^:])\/\/[^\s\\/]+(?:@(?:\d+|ssl))?(?:[\\/]|$|\s)`)
	reUNCMixed1    = regexp.MustCompile(`\/\\{2,}[^\s\\/]`)
	reUNCMixed2    = regexp.MustCompile(`\\{2,}\/[^\s\\/]`)
)

// ContainsVulnerableUncPath mirrors readOnlyCommandValidation.ts containsVulnerableUncPath (Windows-only).
func ContainsVulnerableUncPath(pathOrCommand string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	if !strings.Contains(pathOrCommand, `\`) && !strings.Contains(pathOrCommand, "//") {
		return false
	}
	if reUNCBackslash.MatchString(pathOrCommand) {
		return true
	}
	if reUNCForward.MatchString(pathOrCommand) {
		return true
	}
	if reUNCMixed1.MatchString(pathOrCommand) || reUNCMixed2.MatchString(pathOrCommand) {
		return true
	}
	return false
}
