package filereadtool

import (
	"fmt"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

// FileReadToolName is FILE_READ_TOOL_NAME (FileReadTool/prompt.ts).
const FileReadToolName = "Read"

// FileUnchangedStub is FILE_UNCHANGED_STUB; used by compact post-compact Read dedup.
const FileUnchangedStub = "File unchanged since last read. The content from the earlier Read tool_result in this conversation is still current — refer to that instead of re-reading."

// MaxLinesToRead is MAX_LINES_TO_READ (prompt.ts).
const MaxLinesToRead = 2000

// Description is DESCRIPTION — short async description() return (FileReadTool.ts).
const Description = "Read a file from the local filesystem."

// LineFormatInstruction is LINE_FORMAT_INSTRUCTION (prompt.ts).
const LineFormatInstruction = "- Results are returned using cat -n format, with line numbers starting at 1"

// OffsetInstructionDefault is OFFSET_INSTRUCTION_DEFAULT (prompt.ts).
const OffsetInstructionDefault = "- You can optionally specify a line offset and limit (especially handy for long files), but it's recommended to read the whole file by not providing these parameters"

// OffsetInstructionTargeted is OFFSET_INSTRUCTION_TARGETED (prompt.ts).
const OffsetInstructionTargeted = "- When you already know which part of the file you need, only read that part. This can be important for larger files."

// RenderReadToolPrompt mirrors prompt.ts renderPromptTemplate + FileReadTool.prompt() assembly (lineFormat + maxSize + offset + PDF branch).
func RenderReadToolPrompt(limits FileReadingLimits, pdfSupported bool) string {
	maxSizeInstruction := ""
	if limits.IncludeMaxSizeInPrompt && limits.MaxSizeBytes > 0 {
		maxSizeInstruction = fmt.Sprintf(". Files larger than %s will return an error; use offset and limit for larger files",
			FormatFileSize(int64(limits.MaxSizeBytes)))
	}
	offsetInstruction := OffsetInstructionDefault
	if limits.TargetedRangeNudge {
		offsetInstruction = OffsetInstructionTargeted
	}
	pdfLine := ""
	if pdfSupported {
		pdfLine = fmt.Sprintf(
			"\n- This tool can read PDF files (.pdf). For large PDFs (more than 10 pages), you MUST provide the pages parameter to read specific page ranges (e.g., pages: \"1-5\"). Reading a large PDF without the pages parameter will fail. Maximum %d pages per request.",
			PDFMaxPagesPerRead,
		)
	}
	return fmt.Sprintf(`Reads a file from the local filesystem. You can access any file directly by using this tool.
Assume this tool is able to read all files on the machine. If the User provides a path to a file assume that path is valid. It is okay to read a file that does not exist; an error will be returned.

Usage:
- The file_path parameter must be an absolute path, not a relative path
- By default, it reads up to %d lines starting from the beginning of the file%s
%s
%s
- This tool allows Claude Code to read images (eg PNG, JPG, etc). When reading an image file the contents are presented visually as Claude Code is a multimodal LLM.%s
- This tool can read Jupyter notebooks (.ipynb files) and returns all cells with their outputs, combining code, text, and visualizations.
- This tool can only read files, not directories. To read a directory, use an ls command via the %s tool.
- You will regularly be asked to read screenshots. If the user provides a path to a screenshot, ALWAYS use this tool to view the file at the path. This tool will work with all temporary file paths.
- If you read a file that exists but has empty contents you will receive a system reminder warning in place of file contents.`,
		MaxLinesToRead,
		maxSizeInstruction,
		offsetInstruction,
		LineFormatInstruction,
		pdfLine,
		bashtool.BashToolName,
	)
}
