package bashtool

import "strings"

// PromptLead is the opening line of BashTool/prompt.ts getSimplePrompt() (before sandbox/git sections).
const PromptLead = "Executes a given bash command and returns its output."

// DescriptionFallback mirrors BashTool.ts async description() when input.description is empty.
const DescriptionFallback = "Run shell command"

// ToolDescription returns API/catalog description; upstream uses per-invocation description or fallback.
func ToolDescription(inputDescription string) string {
	if s := strings.TrimSpace(inputDescription); s != "" {
		return s
	}
	return DescriptionFallback
}
