package bashtool

// ValidateSegmentedPipePermissions aliases bashCommandHelpers.ts segmented pipe permission preflight
// (multiple pure-cd pipe segments; cd + git across pipes).
func ValidateSegmentedPipePermissions(command string) error {
	return ValidatePipePermissionPreflight(command)
}
