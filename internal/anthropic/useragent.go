package anthropic

import (
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

const defaultUserAgent = "rabbit-code/phase4"

// UserAgent returns the HTTP User-Agent for Anthropic API calls (utils/http.ts getUserAgent parity).
// Override with RABBIT_CODE_USER_AGENT (mirrors CLAUDE_CODE-side env when present in product).
func UserAgent() string {
	if v := strings.TrimSpace(os.Getenv(features.EnvHTTPUserAgent)); v != "" {
		return v
	}
	return defaultUserAgent
}
