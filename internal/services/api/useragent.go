package anthropic

import "github.com/2456868764/rabbit-code/internal/features"

// UserAgent returns the HTTP User-Agent for Anthropic API calls (utils/http.ts getUserAgent parity).
// Delegates to features.HTTPUserAgent (RABBIT_CODE_USER_AGENT / DefaultHTTPUserAgent).
func UserAgent() string {
	return features.HTTPUserAgent()
}
