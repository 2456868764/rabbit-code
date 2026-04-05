// Package notebookedittool implements the NotebookEdit tool (claude-code-sourcemap/restored-src/src/tools/NotebookEditTool/NotebookEditTool.ts).
//
// TS file mapping: NotebookEditTool.ts → notebook_edit_tool.go; UI.tsx → ui.go (mapToolResult strings; TUI-only exports are headless-deferred); prompt.ts → prompt.go; constants.ts → constants.go.
//
// Parity notes: parseCellId lives in src/utils/notebook.ts → parse_cell_id.go. Read-before-edit and mtime staleness match FileEditTool/FileWriteTool; UNC skips local validate (TS validateInput); json.MarshalIndent(..., "", " ") matches IPYNB_INDENT=1.
package notebookedittool
