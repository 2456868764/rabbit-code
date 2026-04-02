package anthropic

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Retry policy defaults aligned with withRetry.ts (BASE_DELAY_MS, DEFAULT_MAX_RETRIES).
const (
	BaseDelayMS = 500
	// FastBaseDelayMS is the exponential backoff floor when Policy.FastRetry is true (withRetry.ts fast mode).
	FastBaseDelayMS = 100
	DefaultMaxAttempts = 10
	// Max529Retries is how many times we may retry after receiving HTTP 529 (withRetry.ts MAX_529_RETRIES).
	Max529Retries = 3
)

// QuerySource tags the caller for withRetry.ts 529 filtering.
type QuerySource string

const (
	// QuerySourceDefault applies the standard Max529Retries budget for HTTP 529.
	QuerySourceDefault QuerySource = ""
	// QuerySourceNo529 does not retry HTTP 529 (fail fast on overload).
	QuerySourceNo529 QuerySource = "no-529"
	// QuerySourceInteractive labels foreground user-driven traffic (rabbit-local; not in TS FOREGROUND_529_RETRY_SOURCES).
	QuerySourceInteractive QuerySource = "interactive"
	// QuerySourceCompaction is the withRetry.ts / querySource string for compaction ("compact").
	QuerySourceCompaction QuerySource = "compact"
	// QuerySourceBackground labels background / housekeeping traffic (rabbit-local; under StrictForeground529, 529 is not retried).
	QuerySourceBackground QuerySource = "background"

	// --- constants aligned with src/constants/querySource + withRetry.ts FOREGROUND_529_RETRY_SOURCES ---

	QuerySourceReplMainThread                        QuerySource = "repl_main_thread"
	QuerySourceReplMainThreadOutputStyleCustom       QuerySource = "repl_main_thread:outputStyle:custom"
	QuerySourceReplMainThreadOutputStyleExplanatory  QuerySource = "repl_main_thread:outputStyle:Explanatory"
	QuerySourceReplMainThreadOutputStyleLearning     QuerySource = "repl_main_thread:outputStyle:Learning"
	QuerySourceSDK                                   QuerySource = "sdk"
	QuerySourceAgentCustom                           QuerySource = "agent:custom"
	QuerySourceAgentDefault                          QuerySource = "agent:default"
	QuerySourceAgentBuiltin                          QuerySource = "agent:builtin"
	QuerySourceHookAgent                             QuerySource = "hook_agent"
	QuerySourceHookPrompt                            QuerySource = "hook_prompt"
	QuerySourceVerificationAgent                     QuerySource = "verification_agent"
	QuerySourceSideQuestion                          QuerySource = "side_question"
	QuerySourceAutoMode                              QuerySource = "auto_mode"
	QuerySourceBashClassifier                        QuerySource = "bash_classifier"
)

// foreground529RetrySources mirrors withRetry.ts FOREGROUND_529_RETRY_SOURCES (BASH_CLASSIFIER branch always included in Go).
var foreground529RetrySources = map[QuerySource]struct{}{
	QuerySourceReplMainThread:                       {},
	QuerySourceReplMainThreadOutputStyleCustom:      {},
	QuerySourceReplMainThreadOutputStyleExplanatory: {},
	QuerySourceReplMainThreadOutputStyleLearning:    {},
	QuerySourceSDK:                                  {},
	QuerySourceAgentCustom:                          {},
	QuerySourceAgentDefault:                         {},
	QuerySourceAgentBuiltin:                         {},
	QuerySourceCompaction:                           {},
	QuerySourceHookAgent:                            {},
	QuerySourceHookPrompt:                           {},
	QuerySourceVerificationAgent:                    {},
	QuerySourceSideQuestion:                         {},
	QuerySourceAutoMode:                             {},
	QuerySourceBashClassifier:                       {},
}

func allows529Retry(pol Policy) bool {
	if pol.QuerySource == QuerySourceNo529 {
		return false
	}
	if !pol.StrictForeground529 {
		return true
	}
	if pol.QuerySource == QuerySourceDefault {
		return true
	}
	_, ok := foreground529RetrySources[pol.QuerySource]
	return ok
}

// Policy configures HTTP retries.
type Policy struct {
	MaxAttempts int
	// Retry529429 when true retries 429/529 and 5xx.
	Retry529429 bool
	Unattended  bool
	// FastRetry uses FastBaseDelayMS backoff floor (env RABBIT_CODE_FAST_RETRY via DefaultPolicy).
	FastRetry bool
	// QuerySource controls whether HTTP 529 participates in Max529Retries (529 still non-retryable when No529).
	QuerySource QuerySource
	// StrictForeground529 when true applies withRetry.ts FOREGROUND_529_RETRY_SOURCES: only Default ("") and listed QuerySource values retry 529; others fail fast on 529 like No529.
	StrictForeground529 bool
}

// DefaultPolicy returns default retry behavior for foreground streams.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:           DefaultMaxAttempts,
		Retry529429:         true,
		Unattended:          features.UnattendedRetryEnabled(),
		FastRetry:           features.FastRetryEnabled(),
		StrictForeground529: features.StrictForeground529Enabled(),
	}
}

func minBackoffMS(fast bool) int {
	if fast {
		return FastBaseDelayMS
	}
	return BaseDelayMS
}

func backoff(attempt int, unattended, fast bool) time.Duration {
	floor := float64(minBackoffMS(fast))
	base := floor * math.Pow(2, float64(attempt))
	if unattended {
		const capMs = 5 * 60 * 1000
		if base > capMs {
			base = capMs
		}
	}
	jitter := 1.0 + (rand.Float64()*0.2 - 0.1)
	ms := base * jitter
	if ms < floor {
		ms = floor
	}
	return time.Duration(ms * float64(time.Millisecond))
}

// BackoffDuration returns the randomized wait before retry attempt attempt+1 (exported for AC4-1b / tests).
func BackoffDuration(attempt int, pol Policy) time.Duration {
	return backoff(attempt, pol.Unattended, pol.FastRetry)
}

// DoRequest executes req with retries on transient HTTP failures. It closes discarded response bodies.
// 529 responses consume the Max529Retries budget separately from the overall attempt cap (withRetry.ts).
func DoRequest(ctx context.Context, rt http.RoundTripper, req *http.Request, pol Policy) (*http.Response, error) {
	n := pol.MaxAttempts
	if n <= 0 {
		n = 1
	}
	var lastStatus int
	left529 := Max529Retries
	for attempt := 0; attempt < n; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt-1, pol.Unattended, pol.FastRetry)):
			}
		}
		resp, err := rt.RoundTrip(req.Clone(ctx))
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if attempt == n-1 {
				return nil, ClassifyRoundTripError(err)
			}
			continue
		}
		if !pol.Retry529429 || !IsRetryableStatus(resp.StatusCode) {
			return resp, nil
		}
		lastStatus = resp.StatusCode
		if resp.StatusCode == 529 {
			if !allows529Retry(pol) {
				resp.Body.Close()
				return nil, ClassifyHTTP(529)
			}
			if left529 <= 0 {
				resp.Body.Close()
				return nil, ClassifyHTTP(529)
			}
			left529--
		}
		resp.Body.Close()
		if attempt == n-1 {
			return nil, ClassifyHTTP(lastStatus)
		}
	}
	return nil, ClassifyHTTP(lastStatus)
}
