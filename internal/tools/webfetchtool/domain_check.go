package webfetchtool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	domainCheckTimeout = 10 * time.Second
	domainCheckTTL     = 5 * time.Minute
	domainCheckMax     = 128
	domainInfoPath     = "/api/web/domain_info"
)

// domainCheckFailedUserMsg mirrors utils.ts DomainCheckFailedError message.
func domainCheckFailedUserMsg(hostname string) string {
	return fmt.Sprintf(
		"Unable to verify if domain %s is safe to fetch. This may be due to network restrictions or enterprise security policies blocking claude.ai.",
		hostname,
	)
}

var (
	domainCheckMu    sync.Mutex
	domainAllowCache = make(map[string]time.Time) // hostname -> expiry; only 'allowed' cached upstream
)

func domainCheckURL(base, hostname string) (string, error) {
	b := strings.TrimSuffix(strings.TrimSpace(base), "/")
	u, err := url.Parse(b + domainInfoPath)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("domain", hostname)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// CheckDomainBlocklist mirrors utils.ts checkDomainBlocklist (GET api.anthropic.com/api/web/domain_info).
func CheckDomainBlocklist(ctx context.Context, apiBase string, hostname string, client *http.Client) error {
	if hostname == "" {
		return fmt.Errorf("%w: empty hostname", ErrDomainCheckFailed)
	}
	domainCheckMu.Lock()
	if exp, ok := domainAllowCache[hostname]; ok && time.Now().Before(exp) {
		domainCheckMu.Unlock()
		return nil
	}
	domainCheckMu.Unlock()

	if client == nil {
		client = &http.Client{Timeout: domainCheckTimeout}
	}
	reqURL, err := domainCheckURL(apiBase, hostname)
	if err != nil {
		return fmt.Errorf("%s: %v: %w", domainCheckFailedUserMsg(hostname), err, ErrDomainCheckFailed)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("%s: %v: %w", domainCheckFailedUserMsg(hostname), err, ErrDomainCheckFailed)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %v: %w", domainCheckFailedUserMsg(hostname), err, ErrDomainCheckFailed)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))

	if resp.StatusCode == http.StatusOK {
		var parsed struct {
			CanFetch bool `json:"can_fetch"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return fmt.Errorf("%s: %v: %w", domainCheckFailedUserMsg(hostname), err, ErrDomainCheckFailed)
		}
		if parsed.CanFetch {
			domainCheckMu.Lock()
			if len(domainAllowCache) >= domainCheckMax {
				domainAllowCache = make(map[string]time.Time)
			}
			domainAllowCache[hostname] = time.Now().Add(domainCheckTTL)
			domainCheckMu.Unlock()
			return nil
		}
		return fmt.Errorf("Claude Code is unable to fetch from %s: %w", hostname, ErrDomainBlocked)
	}

	return fmt.Errorf("%s (domain check returned status %d): %w", domainCheckFailedUserMsg(hostname), resp.StatusCode, ErrDomainCheckFailed)
}

// ClearDomainAllowCacheForTest clears the positive domain cache (TTL cache).
func ClearDomainAllowCacheForTest() {
	domainCheckMu.Lock()
	domainAllowCache = make(map[string]time.Time)
	domainCheckMu.Unlock()
}
