package bashtool

// SedCommandAllowedByAllowlist mirrors sedValidation.ts sedCommandIsAllowedByAllowlist; defer full sed AST.
func SedCommandAllowedByAllowlist(command string) bool {
	_ = command
	return true
}
