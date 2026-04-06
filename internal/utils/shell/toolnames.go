// Package shell holds shell-related utilities; SHELL_TOOL_NAMES mirrors
// restored-src/src/utils/shell/shellToolUtils.ts.
package shell

import "github.com/2456868764/rabbit-code/internal/tools/powershelltool"

// ShellToolNames returns a copy of SHELL_TOOL_NAMES (Bash, PowerShell order).
func ShellToolNames() []string {
	// Literal "Bash" matches bashtool.BashToolName; avoid importing bashtool (import cycle: bashtool → memdir → query → … → shell).
	return []string{"Bash", powershelltool.PowerShellToolName}
}
