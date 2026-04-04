// Package filereadtool mirrors restored-src/src/tools/FileReadTool/prompt.ts (exported tool name only).
package filereadtool

// FileReadToolName is FILE_READ_TOOL_NAME upstream.
const FileReadToolName = "Read"

// FileUnchangedStub is FILE_UNCHANGED_STUB (FileReadTool/prompt.ts); used by compact post-compact Read dedup.
const FileUnchangedStub = "File unchanged since last read. The content from the earlier Read tool_result in this conversation is still current — refer to that instead of re-reading."
