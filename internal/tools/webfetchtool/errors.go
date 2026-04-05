package webfetchtool

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrDomainBlocked mirrors DomainBlockedError (domain_info returned can_fetch false).
	ErrDomainBlocked = errors.New("webfetchtool: domain blocked")
	// ErrDomainCheckFailed mirrors DomainCheckFailedError (preflight unreachable / non-200).
	ErrDomainCheckFailed = errors.New("webfetchtool: domain check failed")
)

// EgressBlockedError returns an error whose message is JSON matching utils.ts EgressBlockedError.
func EgressBlockedError(domain string) error {
	b, _ := json.Marshal(map[string]string{
		"error_type": "EGRESS_BLOCKED",
		"domain":     domain,
		"message":    fmt.Sprintf("Access to %s is blocked by the network egress proxy.", domain),
	})
	return fmt.Errorf("webfetchtool: %s", string(b))
}
