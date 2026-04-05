package webfetchtool

import (
	"fmt"
	"net/url"
	"strings"
)

func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	switch h {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func parseAndUpgradeURL(s string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil, err
	}
	// Upstream upgrades all http→https; we skip loopback so httptest and local dev work over HTTP.
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		u.Scheme = "https"
	}
	return u, nil
}

// ValidateURL mirrors utils.ts validateURL.
func ValidateURL(s string) error {
	if len(s) > maxURLLength {
		return fmt.Errorf("webfetchtool: URL exceeds max length")
	}
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return fmt.Errorf("webfetchtool: invalid URL: %w", err)
	}
	if u.User != nil {
		return fmt.Errorf("webfetchtool: URL must not include username or password")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("webfetchtool: missing hostname")
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return fmt.Errorf("webfetchtool: hostname must have at least two labels")
	}
	return nil
}
