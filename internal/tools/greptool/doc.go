// Package greptool implements the Grep tool (claude-code-sourcemap/restored-src/src/tools/GrepTool/GrepTool.ts).
//
// TS file mapping: GrepTool.ts → grep_tool.go; UI.tsx → ui.go; prompt.ts → prompt.go.
//
// Parity notes: NODE_ENV=test → files_with_matches sorted by path (GrepTool.ts test branch); negative head_limit → DEFAULT_HEAD_LIMIT like semanticNumber discard; Windows-safe path:line split in content mode (splitFirstRgPathColon over TS indexOf for Windows drive letters).
package greptool
