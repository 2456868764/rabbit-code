package bashtool

import (
	"regexp"
	"strconv"
	"strings"
)

// SearchReadKind mirrors BashTool.tsx isSearchOrReadBashCommand return shape.
type SearchReadKind struct {
	IsSearch bool
	IsRead   bool
	IsList   bool
}

var (
	bashSearchCommands = map[string]struct{}{
		"find": {}, "grep": {}, "rg": {}, "ag": {}, "ack": {}, "locate": {}, "which": {}, "whereis": {},
	}
	bashReadCommands = map[string]struct{}{
		"cat": {}, "head": {}, "tail": {}, "less": {}, "more": {},
		"wc": {}, "stat": {}, "file": {}, "strings": {},
		"jq": {}, "awk": {}, "cut": {}, "sort": {}, "uniq": {}, "tr": {},
	}
	bashListCommands = map[string]struct{}{
		"ls": {}, "tree": {}, "du": {},
	}
	bashSemanticNeutral = map[string]struct{}{
		"echo": {}, "printf": {}, "true": {}, "false": {}, ":": {},
	}
	sleepBlockedLeading = regexp.MustCompile(`^sleep\s+(\d+)\s*$`)
)

// IsSearchOrReadBashCommand mirrors exported isSearchOrReadBashCommand in BashTool.tsx.
func IsSearchOrReadBashCommand(command string) SearchReadKind {
	zero := SearchReadKind{}
	partsWithOperators := SplitCommandWithOperators(command)
	if len(partsWithOperators) == 0 {
		return zero
	}
	var hasSearch, hasRead, hasList, hasNonNeutral bool
	skipNextAsRedirectTarget := false
	for _, part := range partsWithOperators {
		if skipNextAsRedirectTarget {
			skipNextAsRedirectTarget = false
			continue
		}
		if part == ">" || part == ">>" || part == ">&" {
			skipNextAsRedirectTarget = true
			continue
		}
		if part == "||" || part == "&&" || part == "|" || part == ";" {
			continue
		}
		base := strings.Fields(strings.TrimSpace(part))
		if len(base) == 0 {
			continue
		}
		baseCommand := base[0]
		if _, ok := bashSemanticNeutral[baseCommand]; ok {
			continue
		}
		hasNonNeutral = true
		_, isPartSearch := bashSearchCommands[baseCommand]
		_, isPartRead := bashReadCommands[baseCommand]
		_, isPartList := bashListCommands[baseCommand]
		if !isPartSearch && !isPartRead && !isPartList {
			return zero
		}
		if isPartSearch {
			hasSearch = true
		}
		if isPartRead {
			hasRead = true
		}
		if isPartList {
			hasList = true
		}
	}
	if !hasNonNeutral {
		return zero
	}
	return SearchReadKind{IsSearch: hasSearch, IsRead: hasRead, IsList: hasList}
}

// DetectBlockedSleepPattern mirrors BashTool.tsx detectBlockedSleepPattern (MONITOR_TOOL validateInput).
func DetectBlockedSleepPattern(command string) string {
	parts := SplitCommandDeprecated(command)
	if len(parts) == 0 {
		return ""
	}
	first := strings.TrimSpace(parts[0])
	m := sleepBlockedLeading.FindStringSubmatch(first)
	if m == nil {
		return ""
	}
	secs, err := strconv.Atoi(m[1])
	if err != nil || secs < 2 {
		return ""
	}
	rest := strings.TrimSpace(strings.Join(parts[1:], " "))
	if rest != "" {
		return "sleep " + m[1] + " followed by: " + rest
	}
	return "standalone sleep " + m[1]
}
