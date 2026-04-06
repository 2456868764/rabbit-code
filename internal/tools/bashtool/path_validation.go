package bashtool

// PathValidationOptions is a placeholder for pathValidation.ts checkPathConstraints (cwd, dangerous removal, cd+write).
// Full TS parity requires ToolPermissionContext and filesystem layout; headless hosts defer to engine policy.
type PathValidationOptions struct {
	// WorkdirRoot if non-empty: reserved for future path allowlist checks.
	WorkdirRoot string
}

// CheckPathConstraints mirrors pathValidation.ts entry shape; always allows in headless until engine wires opts.
func CheckPathConstraints(command string, _ PathValidationOptions) error {
	_ = command
	return nil
}
