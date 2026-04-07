package readonlycmd

// SedAllowlistCheck is set from bashtool.init to sedValidation.sedCommandIsAllowedByAllowlist equivalent.
var SedAllowlistCheck func(command string) bool
