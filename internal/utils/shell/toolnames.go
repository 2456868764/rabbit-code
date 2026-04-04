// Package shell holds shell-related utilities; SHELL_TOOL_NAMES mirrors
// restored-src/src/utils/shell/shellToolUtils.ts.
package shell

import (
	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
	"github.com/2456868764/rabbit-code/internal/tools/powershelltool"
)

// ShellToolNames returns a copy of SHELL_TOOL_NAMES (Bash, PowerShell order).
func ShellToolNames() []string {
	return []string{bashtool.BashToolName, powershelltool.PowerShellToolName}
}
