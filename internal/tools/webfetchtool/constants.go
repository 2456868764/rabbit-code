package webfetchtool

// Mirrors WebFetchTool/utils.ts resource limits.
const (
	maxURLLength           = 2000
	maxHTTPContentLength   = 10 * 1024 * 1024
	// HTTP client timeout: 60s (FETCH_TIMEOUT_MS upstream).
	maxRedirects           = 10
	maxMarkdownLength      = 100_000
	maxResultSizeChars     = 100_000 // Tool.ts maxResultSizeChars
	webFetchUserAgentLine  = "Claude-User (RabbitCode; +https://support.anthropic.com/)"
)
