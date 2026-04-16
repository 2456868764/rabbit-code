package bashtool

import (
	"strconv"
	"strings"
)

// CommandSemantic mirrors commandSemantics.ts CommandSemantic (function type).
type CommandSemantic func(exitCode int, stdout, stderr string) (isError bool, message string)

// InterpretCommandResult mirrors commandSemantics.ts interpretCommandResult (exit-code semantics for grep/rg/find/diff/test/[).
func InterpretCommandResult(command string, exitCode int, stdout, stderr string) (isError bool, message string) {
	sem := getCommandSemantic(command)
	return sem(exitCode, stdout, stderr)
}

type commandSemantic = CommandSemantic

func defaultSemantic(exitCode int, _, _ string) (bool, string) {
	if exitCode == 0 {
		return false, ""
	}
	return true, "Command failed with exit code " + strconv.Itoa(exitCode)
}

func grepLikeSemantic(exitCode int, _, _ string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, ""
	case exitCode == 1:
		return false, "No matches found"
	default:
		return true, "Command failed with exit code " + strconv.Itoa(exitCode)
	}
}

func findSemantic(exitCode int, _, _ string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, ""
	case exitCode == 1:
		return false, "Some directories were inaccessible"
	default:
		return true, "Command failed with exit code " + strconv.Itoa(exitCode)
	}
}

func diffSemantic(exitCode int, _, _ string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, ""
	case exitCode == 1:
		return false, "Files differ"
	default:
		return true, "Command failed with exit code " + strconv.Itoa(exitCode)
	}
}

func testSemantic(exitCode int, _, _ string) (bool, string) {
	switch {
	case exitCode == 0:
		return false, ""
	case exitCode == 1:
		return false, "Condition is false"
	default:
		return true, "Command failed with exit code " + strconv.Itoa(exitCode)
	}
}

func heuristicallyExtractBaseCommand(command string) string {
	segments := SplitCommandDeprecated(command)
	last := command
	if len(segments) > 0 {
		last = segments[len(segments)-1]
	}
	last = strings.TrimSpace(last)
	if last == "" {
		return ""
	}
	return strings.Fields(last)[0]
}

func getCommandSemantic(command string) commandSemantic {
	base := heuristicallyExtractBaseCommand(command)
	switch base {
	case "grep", "rg":
		return grepLikeSemantic
	case "find":
		return findSemantic
	case "diff":
		return diffSemantic
	case "test", "[":
		return testSemantic
	default:
		return defaultSemantic
	}
}
