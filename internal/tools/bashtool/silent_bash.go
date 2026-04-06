package bashtool

import "strings"

// bashSilentCommands mirrors BashTool.tsx BASH_SILENT_COMMANDS.
var bashSilentCommands = map[string]struct{}{
	"mv": {}, "cp": {}, "rm": {}, "mkdir": {}, "rmdir": {}, "chmod": {}, "chown": {}, "chgrp": {},
	"touch": {}, "ln": {}, "cd": {}, "export": {}, "unset": {}, "wait": {},
}

// IsSilentBashCommand mirrors BashTool.tsx isSilentBashCommand (noOutputExpected).
func IsSilentBashCommand(command string) bool {
	parts := SplitCommandWithOperators(command)
	if len(parts) == 0 {
		return false
	}
	var hasNonFallback bool
	var lastOperator string
	skipNextAsRedirectTarget := false
	for _, part := range parts {
		if skipNextAsRedirectTarget {
			skipNextAsRedirectTarget = false
			continue
		}
		if part == ">" || part == ">>" || part == ">&" {
			skipNextAsRedirectTarget = true
			continue
		}
		if part == "||" || part == "&&" || part == "|" || part == ";" {
			lastOperator = part
			continue
		}
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		baseCommand := fields[0]
		if lastOperator == "||" {
			if _, ok := bashSemanticNeutral[baseCommand]; ok {
				continue
			}
		}
		hasNonFallback = true
		if _, ok := bashSilentCommands[baseCommand]; !ok {
			return false
		}
	}
	return hasNonFallback
}
