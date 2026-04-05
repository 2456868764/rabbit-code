// Package webfetchtool mirrors restored-src/src/tools/WebFetchTool/WebFetchTool.ts and prompt.ts.
package webfetchtool

// WebFetchToolName is WEB_FETCH_TOOL_NAME upstream.
const WebFetchToolName = "WebFetch"

// Description mirrors prompt.ts DESCRIPTION (tool listing / search body).
const Description = `
- Fetches content from a specified URL and processes it using an AI model
- Takes a URL and a prompt as input
- Fetches the URL content, converts HTML to plain text (upstream uses HTML→markdown)
- Processes the content with the prompt using a small, fast model when wired via RunContext.ApplyPrompt; otherwise returns the same structured prompt text upstream sends to Haiku
- Returns the model's response about the content (or the structured prompt in headless mode)
- Use this tool when you need to retrieve and analyze web content

Usage notes:
  - IMPORTANT: If an MCP-provided web fetch tool is available, prefer using that tool instead of this one, as it may have fewer restrictions.
  - The URL must be a fully-formed valid URL
  - HTTP URLs will be automatically upgraded to HTTPS
  - The prompt should describe what information you want to extract from the page
  - This tool is read-only and does not modify any files
  - When a URL redirects to a different host, the tool will inform you and provide the redirect URL; make a new WebFetch request with the redirect URL.
  - Domain blocklist preflight calls https://api.anthropic.com/api/web/domain_info (set RABBIT_CODE_SKIP_WEBFETCH_PREFLIGHT=1 or RunContext.SkipWebFetchPreflight to skip, same as settings skipWebFetchPreflight).
  - Fetched pages are cached for 15 minutes with a 50MB total budget (URL_CACHE upstream).
`
