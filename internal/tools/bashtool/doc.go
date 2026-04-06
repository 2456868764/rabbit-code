// Package bashtool implements the Bash tool (claude-code-sourcemap/restored-src/src/tools/BashTool/BashTool.tsx).
//
// TS mapping: BashTool.tsx → bash_tool.go; commandSemantics.ts → command_semantics.go; background spawn → background.go; silent isSilentBashCommand → silent_bash.go; toolName.ts → toolname.go; prompt.ts → prompt.go (GetSimplePrompt sans SandboxManager/git sections); UI.tsx → ui.go; utils/bash/commands.ts → commands.go; isSearchOrRead/detectBlockedSleep → search_read.go.
// Timeouts: utils/timeouts.ts → timeouts.go. MONITOR sleep gate: RABBIT_MONITOR_TOOL. Explicit run_in_background → goroutine + merged output file (backgroundTaskId / backgroundTaskOutputPath); no task queue / assistant auto-background / Kairos (defer).
// Deferred vs TS: bashPermissions, readOnlyValidation, pathValidation, sandbox, sed, LocalShellTask parity with full TUI/task notifications (see PARITY_H9_BASH_PERMISSIONS.md).
package bashtool
