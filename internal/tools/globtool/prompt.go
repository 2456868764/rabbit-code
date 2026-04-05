// Package globtool mirrors restored-src/src/tools/GlobTool/ (GlobTool.ts, prompt.ts, UI.tsx → glob_tool.go, prompt.go, ui.go).
package globtool

// GlobToolName is GLOB_TOOL_NAME upstream (prompt.ts).
const GlobToolName = "Glob"

// Description mirrors prompt.ts DESCRIPTION (tool schema / prompt body).
const Description = `- Fast file pattern matching tool that works with any codebase size
- Supports glob patterns like "**/*.js" or "src/**/*.ts"
- Returns matching file paths sorted by modification time
- Use this tool when you need to find files by name patterns
- When you are doing an open ended search that may require multiple rounds of globbing and grepping, use the Agent tool instead`
