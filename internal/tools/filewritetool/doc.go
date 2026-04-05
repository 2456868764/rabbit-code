// Package filewritetool implements Write (claude-code-sourcemap/restored-src/src/tools/FileWriteTool/FileWriteTool.ts).
//
// TS file mapping: FileWriteTool.ts → file_write_tool.go; prompt.ts → prompt.go; UI.tsx → ui.go.
// Encoding and line endings follow utils/fileRead.ts readFileSyncWithMetadata + utils/file.ts writeTextContent (utf8 / utf-16 LE BOM sniff, CRLF vs LF).
package filewritetool
