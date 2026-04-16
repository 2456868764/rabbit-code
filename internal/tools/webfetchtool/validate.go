package webfetchtool

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateInputResult mirrors the validateInput return shape in WebFetchTool.ts.
type ValidateInputResult struct {
	Result    bool
	Message   string
	Meta      map[string]string
	ErrorCode int
}

// ValidateInput mirrors WebFetchTool.ts validateInput(input): only checks URL parseability.
// Returns { result: true } on success or { result: false, message: ..., meta: {reason}, errorCode: 1 } on failure.
func ValidateInput(rawURL string) ValidateInputResult {
	if _, err := url.Parse(rawURL); err != nil || rawURL == "" {
		return ValidateInputResult{
			Result:    false,
			Message:   fmt.Sprintf("Error: Invalid URL %q. The URL provided could not be parsed.", rawURL),
			Meta:      map[string]string{"reason": "invalid_url"},
			ErrorCode: 1,
		}
	}
	return ValidateInputResult{Result: true}
}

func parseAndUpgradeURL(s string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil, err
	}
	if u.Scheme == "http" {
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
