package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

// OAuthAPIBase returns the OAuth console/API base URL (constants/oauth.ts + usage.ts `${getOauthConfig().BASE_API_URL}`).
func OAuthAPIBase() string {
	if v := strings.TrimSpace(os.Getenv(features.EnvOAuthBaseURL)); v != "" {
		return strings.TrimRight(v, "/")
	}
	// Public default; staging/local via env (align USE_STAGING_OAUTH in TS).
	return "https://console.anthropic.com"
}

// RateLimit mirrors usage.ts RateLimit.
type RateLimit struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    *string  `json:"resets_at"`
}

// ExtraUsage mirrors usage.ts ExtraUsage.
type ExtraUsage struct {
	IsEnabled    bool     `json:"is_enabled"`
	MonthlyLimit *float64 `json:"monthly_limit"`
	UsedCredits  *float64 `json:"used_credits"`
	Utilization  *float64 `json:"utilization"`
}

// Utilization mirrors fetchUtilization() return (usage.ts).
type Utilization struct {
	FiveHour          *RateLimit  `json:"five_hour"`
	SevenDay          *RateLimit  `json:"seven_day"`
	SevenDayOAuthApps *RateLimit  `json:"seven_day_oauth_apps"`
	SevenDayOpus      *RateLimit  `json:"seven_day_opus"`
	SevenDaySonnet    *RateLimit  `json:"seven_day_sonnet"`
	ExtraUsage        *ExtraUsage `json:"extra_usage"`
}

// FetchUtilization performs GET {oauthBase}/api/oauth/usage with Bearer auth (services/api/usage.ts).
// skipSubscriberCheck: when true, skips the TS isClaudeAISubscriber/hasProfileScope gate (for tests/integration).
func FetchUtilization(ctx context.Context, rt http.RoundTripper, oauthBase, bearer string, skipSubscriberCheck bool) (*Utilization, error) {
	if !skipSubscriberCheck {
		// TS returns {} without calling API — caller should set skip in tests or wire real auth state in Phase 10.
		if strings.TrimSpace(bearer) == "" {
			return &Utilization{}, nil
		}
	}
	if rt == nil {
		rt = http.DefaultTransport
	}
	u := strings.TrimRight(oauthBase, "/") + "/api/oauth/usage"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent())
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req = req.WithContext(cctx)
	// Align transient retries with Messages (429/529/5xx) via DoRequest (usage.ts + withRetry parity).
	pol := Policy{
		MaxAttempts:         4,
		Retry529429:         true,
		Unattended:          features.UnattendedRetryEnabled(),
		FastRetry:           features.FastRetryEnabled(),
		StrictForeground529: features.StrictForeground529Enabled(),
	}
	resp, err := DoRequest(cctx, rt, req, pol)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth usage: status %d: %s", resp.StatusCode, string(body))
	}
	var out Utilization
	if len(body) == 0 || string(body) == "{}" {
		return &out, nil
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
