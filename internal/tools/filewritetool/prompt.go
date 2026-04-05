package filewritetool

import (
	"fmt"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

// FileWriteToolName is FILE_WRITE_TOOL_NAME (FileWriteTool/prompt.ts).
const FileWriteToolName = "Write"

// Description is the short async description() return from FileWriteTool.ts (distinct from prompt body).
const Description = "Write a file to the local filesystem."

func preReadInstruction() string {
	return fmt.Sprintf("\n- If this is an existing file, you MUST use the %s tool first to read the file's contents. This tool will fail if you did not read the file first.", filereadtool.FileReadToolName)
}

// GetWriteToolDescription mirrors FileWriteTool/prompt.ts getWriteToolDescription (async prompt() body).
func GetWriteToolDescription() string {
	return `Writes a file to the local filesystem.

Usage:
- This tool will overwrite the existing file if there is one at the provided path.` + preReadInstruction() + `
- Prefer the Edit tool for modifying existing files — it only sends the diff. Only use this tool to create new files or for complete rewrites.
- NEVER create documentation files (*.md) or README files unless explicitly requested by the User.
- Only use emojis if the user explicitly requests it. Avoid writing emojis to files unless asked.`
}
