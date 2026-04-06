package bashtool

// BashCommandSafeForAutoRun mirrors bashSecurity.ts optimistic path: headless engine does not run AST security classifier.
// When false, callers may escalate to ask/deny in a future permission hook.
func BashCommandSafeForAutoRun(command string) bool {
	_ = command
	return true
}
