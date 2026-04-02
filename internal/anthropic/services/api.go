// Package services maps restored-src/src/services/api/*.ts to HTTP request shapes for Phase 4 (AC4-7).
// Paths are best-effort parity for mock servers; refine against each TS module as integration grows.
package services

import (
	"fmt"
	"net/http"
	"strings"
)

// TSFile names under services/api/ (§4.1).
const (
	AdminRequests             = "adminRequests.ts"
	Bootstrap                 = "bootstrap.ts"
	Client                    = "client.ts"
	Claude                    = "claude.ts"
	DumpPrompts               = "dumpPrompts.ts"
	EmptyUsage                = "emptyUsage.ts"
	ErrorUtils                = "errorUtils.ts"
	Errors                    = "errors.ts"
	FilesAPI                  = "filesApi.ts"
	FirstTokenDate            = "firstTokenDate.ts"
	Grove                     = "grove.ts"
	Logging                   = "logging.ts"
	MetricsOptOut             = "metricsOptOut.ts"
	OverageCreditGrant        = "overageCreditGrant.ts"
	PromptCacheBreakDetection = "promptCacheBreakDetection.ts"
	Referral                  = "referral.ts"
	SessionIngress            = "sessionIngress.ts"
	UltrareviewQuota          = "ultrareviewQuota.ts"
	Usage                     = "usage.ts"
	WithRetry                 = "withRetry.ts"
)

// AllTSFiles is the full §4.1 file list for coverage tests.
var AllTSFiles = []string{
	AdminRequests, Bootstrap, Client, Claude, DumpPrompts, EmptyUsage, ErrorUtils, Errors,
	FilesAPI, FirstTokenDate, Grove, Logging, MetricsOptOut, OverageCreditGrant,
	PromptCacheBreakDetection, Referral, SessionIngress, UltrareviewQuota, Usage, WithRetry,
}

// Builder constructs a probe HTTP request against base API URL (no body unless noted).
type Builder func(baseURL, oauthBase string) (*http.Request, error)

// Builders maps every TS file to a builder (AC4-7).
var Builders = map[string]Builder{
	AdminRequests: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/internal/admin/health", nil)
	},
	Bootstrap: func(_, oauth string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(oauth)+"/api/bootstrap", nil)
	},
	Client: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/messages", nil)
	},
	Claude: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/messages", nil)
	},
	DumpPrompts: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/internal/dump_prompts", nil)
	},
	EmptyUsage: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/usage/empty", nil)
	},
	ErrorUtils: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/errors/utils", nil)
	},
	Errors: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/errors/classify", nil)
	},
	FilesAPI: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/files", nil)
	},
	FirstTokenDate: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/first_token_date", nil)
	},
	Grove: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/grove", nil)
	},
	Logging: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/log", nil)
	},
	MetricsOptOut: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/metrics/opt_out", nil)
	},
	OverageCreditGrant: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/overage/credit_grant", nil)
	},
	PromptCacheBreakDetection: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/prompt_cache/break_detection", nil)
	},
	Referral: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/referral", nil)
	},
	SessionIngress: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/session/ingress", nil)
	},
	UltrareviewQuota: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(base)+"/v1/ultrareview/quota", nil)
	},
	Usage: func(_, oauth string) (*http.Request, error) {
		return http.NewRequest(http.MethodGet, trim(oauth)+"/api/oauth/usage", nil)
	},
	WithRetry: func(base, _ string) (*http.Request, error) {
		return http.NewRequest(http.MethodPost, trim(base)+"/v1/messages", nil)
	},
}

func trim(s string) string { return strings.TrimRight(s, "/") }

// HasTSModule reports whether tsFile is a registered services/api builder (e.g. emptyUsage.ts).
func HasTSModule(tsFile string) bool {
	_, ok := Builders[tsFile]
	return ok
}

// BuildRequest returns a probe request for the named TS file.
func BuildRequest(tsFile, anthropicBase, oauthBase string) (*http.Request, error) {
	b, ok := Builders[tsFile]
	if !ok {
		return nil, fmt.Errorf("services/api: no builder for %q", tsFile)
	}
	return b(anthropicBase, oauthBase)
}
