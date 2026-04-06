// Package bashtool implements the Bash tool (claude-code-sourcemap/restored-src/src/tools/BashTool/BashTool.tsx).
//
// TS mapping: BashTool.tsx → bash_tool.go; toolName.ts → toolname.go; prompt.ts → prompt.go (GetSimplePrompt sans SandboxManager/git sections); UI.tsx → ui.go; utils/bash/commands.ts split paths → commands.go; BashTool.tsx isSearchOrRead/detectBlockedSleep → search_read.go.
// Timeouts: utils/timeouts.ts → timeouts.go. MONITOR sleep gate: features.MonitorToolEnabled (RABBIT_MONITOR_TOOL) ↔ feature('MONITOR_TOOL').
// Deferred vs TS: bashPermissions/readOnlyValidation/sed/LocalShellTask/background execution/sandbox AST (see PARITY_H9_BASH_PERMISSIONS.md).
package bashtool
