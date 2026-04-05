package webfetchtool

import (
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// defaultFetchInnerUserAgent matches services/api/useragent.go defaultUserAgent (no import — avoids api→compact→webfetch cycle).
const defaultFetchInnerUserAgent = "rabbit-code/api"

// WebFetchUserAgent mirrors utils/http.ts getWebFetchUserAgent (Claude-User + same inner UA as anthropic.UserAgent()).
func WebFetchUserAgent() string {
	ua := strings.TrimSpace(os.Getenv(features.EnvHTTPUserAgent))
	if ua == "" {
		ua = defaultFetchInnerUserAgent
	}
	return "Claude-User (" + ua + "; +https://support.anthropic.com/)"
}
