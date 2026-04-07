package bashtool

import (
	"strings"

	"github.com/2456868764/rabbit-code/internal/extractbash"
	"github.com/2456868764/rabbit-code/internal/readonlycmd"
)

func init() {
	readonlycmd.SedAllowlistCheck = SedCommandAllowedByAllowlist
}

// ReadOnlyCommandLineAllowed is true when every pipe segment and each &&/;/|| subcommand passes
// readonlycmd allowlist + validateFlags (readOnlyCommandValidation) OR extractbash read-only subset.
func ReadOnlyCommandLineAllowed(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return true
	}
	for _, seg := range shellSegments(command) {
		for _, sub := range SplitCommandDeprecated(seg) {
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			toks, err := readonlycmd.TokenizeShellWords(sub)
			if err != nil {
				if extractbash.IsReadOnlyShellCommand(sub) {
					continue
				}
				return false
			}
			if readonlycmd.IsCommandSafeViaFlagParsing(sub, toks) {
				continue
			}
			if extractbash.IsReadOnlyShellCommand(sub) {
				continue
			}
			return false
		}
	}
	return true
}
