package webfetchtool

// redirectCodeText mirrors WebFetchTool.ts statusText for cross-host redirect messages (302 default "Found").
func redirectCodeText(code int) string {
	switch code {
	case 301:
		return "Moved Permanently"
	case 308:
		return "Permanent Redirect"
	case 307:
		return "Temporary Redirect"
	default:
		return "Found"
	}
}
