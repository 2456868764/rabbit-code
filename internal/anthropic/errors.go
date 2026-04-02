package anthropic

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Kind classifies API failures for query transitions (services/api/errors.ts).
type Kind string

const (
	KindUnknown          Kind = "unknown"
	KindPromptTooLong    Kind = "prompt_too_long"
	KindMaxOutputTokens  Kind = "max_output_tokens"
	KindRateLimit        Kind = "rate_limit"
	KindOverloaded       Kind = "overloaded" // 529
	KindConnection       Kind = "connection"
	KindUnauthorized     Kind = "unauthorized"
)

// APIError is a typed error with optional HTTP status and structured kind.
type APIError struct {
	Kind   Kind
	Status int
	Msg    string
}

func (e *APIError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("anthropic API: %s (status %d): %s", e.Kind, e.Status, e.Msg)
	}
	return fmt.Sprintf("anthropic API: %s: %s", e.Kind, e.Msg)
}

// ClassifyHTTP maps status to Kind (partial parity with SDK APIError).
func ClassifyHTTP(status int) *APIError {
	switch status {
	case http.StatusTooManyRequests:
		return &APIError{Kind: KindRateLimit, Status: status, Msg: "rate limited"}
	case 529:
		return &APIError{Kind: KindOverloaded, Status: status, Msg: "overloaded"}
	case http.StatusUnauthorized:
		return &APIError{Kind: KindUnauthorized, Status: status, Msg: "unauthorized"}
	default:
		if status >= 500 {
			return &APIError{Kind: KindUnknown, Status: status, Msg: "server error"}
		}
		return &APIError{Kind: KindUnknown, Status: status, Msg: "request failed"}
	}
}

// ErrPromptTooLongMessage is the sentinel prefix from PROMPT_TOO_LONG_ERROR_MESSAGE.
const ErrPromptTooLongMessage = "Prompt is too long"

var ptlRe = regexp.MustCompile(`(?i)prompt is too long[^0-9]*(\d+)\s*tokens?\s*>\s*(\d+)`)

// ParsePromptTooLongTokenCounts mirrors parsePromptTooLongTokenCounts (errors.ts).
func ParsePromptTooLongTokenCounts(raw string) (actual, limit int64, ok bool) {
	m := ptlRe.FindStringSubmatch(raw)
	if len(m) != 3 {
		return 0, 0, false
	}
	a, _ := strconv.ParseInt(m[1], 10, 64)
	l, _ := strconv.ParseInt(m[2], 10, 64)
	return a, l, true
}

// ClassifyBody inspects JSON error message text for prompt_too_long style strings.
func ClassifyBody(msg string) Kind {
	s := strings.ToLower(msg)
	if strings.Contains(s, "prompt is too long") {
		return KindPromptTooLong
	}
	if strings.Contains(s, "max output tokens") || strings.Contains(s, "max_output_tokens") {
		return KindMaxOutputTokens
	}
	return KindUnknown
}

// IsRetryableStatus returns true for 429, 529, and 5xx (withRetry.ts transient capacity).
func IsRetryableStatus(code int) bool {
	if code == 429 || code == 529 {
		return true
	}
	return code >= 500 && code < 600
}

// ErrAborted is returned when the request context is cancelled.
var ErrAborted = errors.New("anthropic: request aborted")

// ErrPromptCacheBreakDetected is returned from ReadAssistantStream when PROMPT_CACHE_BREAK_DETECTION
// is enabled and an SSE error payload matches prompt cache break heuristics (promptCacheBreakDetection.ts).
var ErrPromptCacheBreakDetected = errors.New("anthropic: prompt cache break detected")

// ClassifyRoundTripError maps transport-layer failures to APIError with KindConnection when appropriate
// (withRetry.ts / errors.ts connection class); otherwise returns err unchanged.
func ClassifyRoundTripError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return &APIError{Kind: KindConnection, Msg: err.Error()}
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &APIError{Kind: KindConnection, Msg: err.Error()}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return &APIError{Kind: KindConnection, Msg: err.Error()}
	}
	return err
}
