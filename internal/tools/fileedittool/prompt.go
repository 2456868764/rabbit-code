package fileedittool

import (
	"fmt"
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

// ShortDescription mirrors FileEditTool.ts async description() return.
const ShortDescription = "A tool for editing files"

func editPreReadInstruction() string {
	return fmt.Sprintf("\n- You must use your `%s` tool at least once in the conversation before editing. This tool will error if you attempt an edit without reading the file. ", filereadtool.FileReadToolName)
}

func linePrefixFormatDescription() string {
	if filereadtool.CompactLinePrefixEnabled() {
		return "line number + tab"
	}
	return "spaces + line number + arrow"
}

func minimalUniquenessHint() string {
	if strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant" {
		return "\n- Use the smallest old_string that's clearly unique — usually 2-4 adjacent lines is sufficient. Avoid including 10+ lines of context when less uniquely identifies the target."
	}
	return ""
}

// GetEditToolPrompt mirrors FileEditTool/prompt.ts getEditToolDescription (async prompt() body).
func GetEditToolPrompt() string {
	bt := "`"
	return fmt.Sprintf(`Performs exact string replacements in files.

Usage:%s
- When editing text from Read tool output, ensure you preserve the exact indentation (tabs/spaces) as it appears AFTER the line number prefix. The line number prefix format is: %s. Everything after that is the actual file content to match. Never include any part of the line number prefix in the old_string or new_string.
- ALWAYS prefer editing existing files in the codebase. NEVER write new files unless explicitly required.
- Only use emojis if the user explicitly requests it. Avoid adding emojis to files unless asked.
- The edit will FAIL if %sold_string%s is not unique in the file. Either provide a larger string with more surrounding context to make it unique or use %sreplace_all%s to change every instance of %sold_string%s.%s
- Use %sreplace_all%s for replacing and renaming strings across the file. This parameter is useful if you want to rename a variable for instance.`,
		editPreReadInstruction(),
		linePrefixFormatDescription(),
		bt, bt, bt, bt, bt, bt,
		minimalUniquenessHint(),
		bt, bt,
	)
}
