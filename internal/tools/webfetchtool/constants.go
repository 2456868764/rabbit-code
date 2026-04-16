package webfetchtool

// Mirrors WebFetchTool/utils.ts resource limits.
const (
	maxURLLength         = 2000
	maxHTTPContentLength = 10 * 1024 * 1024
	// HTTP client timeout: 60s (FETCH_TIMEOUT_MS upstream).
	maxRedirects = 10

	// MaxMarkdownLength mirrors utils.ts MAX_MARKDOWN_LENGTH (exported for callers).
	MaxMarkdownLength = 100_000
	// MaxResultSizeChars mirrors WebFetchTool.ts maxResultSizeChars (exported for callers).
	MaxResultSizeChars = 100_000

	// DefaultDomainCheckBaseURL is the origin for /api/web/domain_info (utils.ts checkDomainBlocklist).
	DefaultDomainCheckBaseURL = "https://api.anthropic.com"
)

// Tool metadata constants from WebFetchTool.ts buildTool definition.
const (
	// SearchHint mirrors searchHint in WebFetchTool.ts.
	SearchHint = "fetch and extract content from a URL"
	// UserFacingName mirrors userFacingName() in WebFetchTool.ts.
	UserFacingName = "Fetch"
	// ShouldDefer mirrors shouldDefer: true in WebFetchTool.ts.
	ShouldDefer = true
	// IsConcurrencySafe mirrors isConcurrencySafe() in WebFetchTool.ts.
	IsConcurrencySafe = true
	// IsReadOnly mirrors isReadOnly() in WebFetchTool.ts.
	IsReadOnly = true
)
