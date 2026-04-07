package bashtool

import (
	"strings"
)

// IsExtractReadOnlyBashInputJSON is true for bashtool read-only mode: COMMAND_ALLOWLIST + validateFlags OR extractbash subset.
func IsExtractReadOnlyBashInputJSON(command string) bool {
	return ReadOnlyCommandLineAllowed(strings.TrimSpace(command))
}
