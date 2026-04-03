package anthropic

import (
	"net/http"
	"os"
	"strings"
)

// EnvAnthropicCustomHeaders is client.ts ANTHROPIC_CUSTOM_HEADERS (curl-style "Name: Value" per line).
const EnvAnthropicCustomHeaders = "ANTHROPIC_CUSTOM_HEADERS"

// EnvRabbitAnthropicCustomHeaders is merged after EnvAnthropicCustomHeaders; same format.
const EnvRabbitAnthropicCustomHeaders = "RABBIT_CODE_ANTHROPIC_CUSTOM_HEADERS"

// ParseCurlStyleHeaders parses newline-separated "Name: Value" lines (client.ts getCustomHeaders).
func ParseCurlStyleHeaders(block string) http.Header {
	h := make(http.Header)
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			continue
		}
		name := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if name != "" {
			h.Set(name, val)
		}
	}
	return h
}

// MergedCustomHeadersFromEnv merges ANTHROPIC_CUSTOM_HEADERS then RABBIT_CODE_ANTHROPIC_CUSTOM_HEADERS (latter wins on duplicate keys).
func MergedCustomHeadersFromEnv() http.Header {
	out := ParseCurlStyleHeaders(os.Getenv(EnvAnthropicCustomHeaders))
	rabbit := ParseCurlStyleHeaders(os.Getenv(EnvRabbitAnthropicCustomHeaders))
	for k, vv := range rabbit {
		if len(vv) > 0 {
			out.Set(k, vv[0])
		}
	}
	return out
}
