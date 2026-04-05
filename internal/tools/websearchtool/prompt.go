// Package websearchtool mirrors restored-src/src/tools/WebSearchTool (prompt.ts, WebSearchTool.ts).
package websearchtool

import (
	"os"
	"strings"
	"time"
)

// WebSearchToolName is WEB_SEARCH_TOOL_NAME upstream.
const WebSearchToolName = "WebSearch"

// localMonthYear mirrors getLocalMonthYear in restored-src/src/constants/common.ts ("Month YYYY", en-US).
func localMonthYear() string {
	var t time.Time
	if s := strings.TrimSpace(os.Getenv("CLAUDE_CODE_OVERRIDE_DATE")); s != "" {
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			t = parsed
		} else if parsed, err := time.Parse("2006-01-02", s); err == nil {
			t = parsed
		}
	}
	if t.IsZero() {
		t = time.Now().In(time.Local)
	}
	return t.Format("January 2006")
}

// PromptBody mirrors getWebSearchPrompt() in prompt.ts (dynamic month/year).
func PromptBody() string {
	currentMonthYear := localMonthYear()
	return `

- Allows Claude to search the web and use the results to inform responses
- Provides up-to-date information for current events and recent data
- Returns search result information formatted as search result blocks, including links as markdown hyperlinks
- Use this tool for accessing information beyond Claude's knowledge cutoff
- Searches are performed automatically within a single API call

CRITICAL REQUIREMENT - You MUST follow this:
  - After answering the user's question, you MUST include a "Sources:" section at the end of your response
  - In the Sources section, list all relevant URLs from the search results as markdown hyperlinks: [Title](URL)
  - This is MANDATORY - never skip including sources in your response
  - Example format:

    [Your answer here]

    Sources:
    - [Source Title 1](https://example.com/1)
    - [Source Title 2](https://example.com/2)

Usage notes:
  - Domain filtering is supported to include or block specific websites
  - Web search is only available in the US

IMPORTANT - Use the correct year in search queries:
  - The current month is ` + currentMonthYear + `. You MUST use this year when searching for recent information, documentation, or current events.
  - Example: If the user asks for "latest React docs", search for "React documentation" with the current year, NOT last year
`
}

// Description is the static listing body (ToolSearch catalog); excludes dynamic month line (prompt.ts listing-style).
const Description = `
- Allows Claude to search the web and use the results to inform responses
- Provides up-to-date information for current events and recent data
- Returns search result information formatted as search result blocks, including links as markdown hyperlinks
- Use this tool for accessing information beyond Claude's knowledge cutoff
- Searches are performed automatically within a single API call

Usage notes:
  - Domain filtering is supported to include or block specific websites
  - Web search is only available in the US
`
