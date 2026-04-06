// Package bashtool implements the Bash tool (claude-code-sourcemap/restored-src/src/tools/BashTool/BashTool.tsx).
//
// TS mapping (incremental): BashTool.tsx → bash_tool.go; toolName.ts → toolname.go; prompt.ts getSimplePrompt/sandbox/git → defer (PromptLead + DescriptionFallback only); UI.tsx → ui.go (mapToolResult string). Other BashTool/*.ts (permissions, sandbox, sed, ast) → Phase 6 / H9 follow-on.
// Timeouts: utils/timeouts.ts → timeouts.go (BASH_DEFAULT_TIMEOUT_MS / BASH_MAX_TIMEOUT_MS).
package bashtool
