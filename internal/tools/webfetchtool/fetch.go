package webfetchtool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type redirectInfo struct {
	OriginalURL string
	RedirectURL string
	StatusCode  int
}

type fetchedRaw struct {
	Body        []byte
	StatusCode  int
	StatusText  string
	ContentType string
	FinalURL    string
}

func stripWww(h string) string {
	return strings.TrimPrefix(strings.ToLower(h), "www.")
}

// isPermittedRedirect mirrors utils.ts isPermittedRedirect (same host modulo www., same scheme/port).
func isPermittedRedirect(originalURL, redirectURL string) bool {
	ou, err1 := url.Parse(originalURL)
	ru, err2 := url.Parse(redirectURL)
	if err1 != nil || err2 != nil {
		return false
	}
	if ru.Scheme != ou.Scheme {
		return false
	}
	if ru.Port() != ou.Port() {
		return false
	}
	if ru.User != nil {
		return false
	}
	return stripWww(ou.Hostname()) == stripWww(ru.Hostname())
}

// responseReasonPhrase mirrors axios response.statusText (HTTP reason after status code).
func responseReasonPhrase(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	code := resp.StatusCode
	s := strings.TrimSpace(resp.Status)
	prefix := strconv.Itoa(code) + " "
	if strings.HasPrefix(s, prefix) {
		if reason := strings.TrimSpace(strings.TrimPrefix(s, prefix)); reason != "" {
			return reason
		}
	}
	return http.StatusText(code)
}

func getWithPermittedRedirects(ctx context.Context, client *http.Client, currentURL string, depth int) (fetchedRaw, *redirectInfo, error) {
	if depth > maxRedirects {
		return fetchedRaw{}, nil, fmt.Errorf("webfetchtool: too many redirects (exceeded %d)", maxRedirects)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, currentURL, nil)
	if err != nil {
		return fetchedRaw{}, nil, err
	}
	req.Header.Set("Accept", "text/markdown, text/html, */*")
	req.Header.Set("User-Agent", WebFetchUserAgent())

	resp, err := client.Do(req)
	if err != nil {
		return fetchedRaw{}, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden && strings.EqualFold(resp.Header.Get("X-Proxy-Error"), "blocked-by-allowlist") {
		u, _ := url.Parse(currentURL)
		host := ""
		if u != nil {
			host = u.Hostname()
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		return fetchedRaw{}, nil, EgressBlockedError(host)
	}

	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		loc := resp.Header.Get("Location")
		if loc == "" {
			return fetchedRaw{}, nil, fmt.Errorf("webfetchtool: redirect missing Location header")
		}
		locURL, err := url.Parse(loc)
		if err != nil {
			return fetchedRaw{}, nil, err
		}
		base, _ := url.Parse(currentURL)
		resolved := base.ResolveReference(locURL).String()
		if isPermittedRedirect(currentURL, resolved) {
			return getWithPermittedRedirects(ctx, client, resolved, depth+1)
		}
		return fetchedRaw{}, &redirectInfo{
			OriginalURL: currentURL,
			RedirectURL: resolved,
			StatusCode:  resp.StatusCode,
		}, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fetchedRaw{}, nil, fmt.Errorf("webfetchtool: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxHTTPContentLength+1)))
	if err != nil {
		return fetchedRaw{}, nil, err
	}
	if len(body) > maxHTTPContentLength {
		return fetchedRaw{}, nil, fmt.Errorf("webfetchtool: response exceeds max content length")
	}

	return fetchedRaw{
		Body:        body,
		StatusCode:  resp.StatusCode,
		StatusText:  responseReasonPhrase(resp),
		ContentType: resp.Header.Get("Content-Type"),
		FinalURL:    currentURL,
	}, nil, nil
}

func defaultFetchClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
