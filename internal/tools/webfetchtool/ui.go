package webfetchtool

import "net/url"

// GetToolUseSummary mirrors getToolUseSummary in UI.tsx — returns the hostname of the URL.
func GetToolUseSummary(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return u.Hostname()
}

// ToolUseDescription mirrors async description(input) in WebFetchTool.ts.
func ToolUseDescription(rawURL string) string {
	hostname := GetToolUseSummary(rawURL)
	if hostname == "" {
		return "Claude wants to fetch content from this URL"
	}
	return "Claude wants to fetch content from " + hostname
}

// ActivityDescription mirrors getActivityDescription(input) in WebFetchTool.ts.
func ActivityDescription(rawURL string) string {
	s := GetToolUseSummary(rawURL)
	if s == "" {
		return "Fetching web page"
	}
	return "Fetching " + s
}

// ToAutoClassifierInput mirrors toAutoClassifierInput in WebFetchTool.ts.
func ToAutoClassifierInput(rawURL, prompt string) string {
	if prompt != "" {
		return rawURL + ": " + prompt
	}
	return rawURL
}
