package webfetchtool

import "github.com/2456868764/rabbit-code/internal/features"

// WebFetchUserAgent mirrors utils/http.ts getWebFetchUserAgent (Claude-User + features.HTTPUserAgent).
func WebFetchUserAgent() string {
	return "Claude-User (" + features.HTTPUserAgent() + "; +https://support.anthropic.com/)"
}
