// Package greptool mirrors restored-src/src/tools/GrepTool/prompt.ts.
package greptool

// GrepToolName is GREP_TOOL_NAME upstream.
const GrepToolName = "Grep"

// GetDescription mirrors getDescription() in prompt.ts (tool description string).
func GetDescription() string {
	return `A powerful search tool built on ripgrep

  Usage:
  - ALWAYS use ` + GrepToolName + ` for search tasks. NEVER invoke ` + "`grep` or `rg`" + ` as a Bash command. The ` + GrepToolName + ` tool has been optimized for correct permissions and access.
  - Supports full regex syntax (e.g., "log.*Error", "function\\s+\\w+")
  - Filter files with glob parameter (e.g., "*.js", "**/*.tsx") or type parameter (e.g., "js", "py", "rust")
  - Output modes: "content" shows matching lines, "files_with_matches" shows only file paths (default), "count" shows match counts
  - Use the Agent tool for open-ended searches requiring multiple rounds
  - Pattern syntax: Uses ripgrep (not grep) - literal braces need escaping (use ` + "`interface\\{\\}`" + ` to find ` + "`interface{}`" + ` in Go code)
  - Multiline matching: By default patterns match within single lines only. For cross-line patterns, use ` + "`multiline: true`" + `
`
}
