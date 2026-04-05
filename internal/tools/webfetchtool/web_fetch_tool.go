package webfetchtool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

// WebFetch implements tools.Tool (WebFetchTool.ts).
type WebFetch struct{}

// New returns a WebFetch tool.
func New() *WebFetch { return &WebFetch{} }

func (w *WebFetch) Name() string { return WebFetchToolName }

func (w *WebFetch) Aliases() []string { return nil }

type webFetchInput struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

type webFetchOutput struct {
	Bytes      int    `json:"bytes"`
	Code       int    `json:"code"`
	CodeText   string `json:"codeText"`
	Result     string `json:"result"`
	DurationMs int64  `json:"durationMs"`
	URL        string `json:"url"`
}

func bytesToUTF8(b []byte) string {
	return strings.ToValidUTF8(string(b), "\uFFFD")
}

func skipWebFetchPreflight(rc *RunContext) bool {
	if rc != nil && rc.SkipWebFetchPreflight != nil {
		return *rc.SkipWebFetchPreflight
	}
	return features.SkipWebFetchPreflight()
}

func toolResultsDir(rc *RunContext) string {
	if rc != nil && rc.ToolResultsDir != "" {
		return rc.ToolResultsDir
	}
	d, err := os.UserCacheDir()
	if err != nil || d == "" {
		d = os.TempDir()
	}
	return filepath.Join(d, "rabbit-code", "tool-results")
}

func domainCheckOrigin(rc *RunContext) string {
	if rc != nil && strings.TrimSpace(rc.DomainCheckBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(rc.DomainCheckBaseURL), "/")
	}
	return DefaultDomainCheckBaseURL
}

// Run fetches a URL and produces result text aligned with WebFetchTool.call.
func (w *WebFetch) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var in webFetchInput
	if err := json.Unmarshal(inputJSON, &in); err != nil {
		return nil, fmt.Errorf("webfetchtool: invalid json: %w", err)
	}
	urlInput := strings.TrimSpace(in.URL)
	prompt := strings.TrimSpace(in.Prompt)
	if urlInput == "" {
		return nil, fmt.Errorf("webfetchtool: missing url")
	}
	if prompt == "" {
		return nil, fmt.Errorf("webfetchtool: missing prompt")
	}
	if err := ValidateURL(urlInput); err != nil {
		return nil, err
	}

	rc := RunContextFrom(ctx)
	client := defaultFetchClient()
	if rc != nil && rc.HTTPClient != nil {
		client = rc.HTTPClient
	}

	var rawBytes int
	var raw fetchedRaw
	var redir *redirectInfo
	var fetchErr error
	var markdown string
	var ct string
	var persistedPath string
	var persistedSize int

	if cached, ok := urlCacheGet(urlInput); ok {
		rawBytes = cached.bytes
		raw = fetchedRaw{
			StatusCode:  cached.code,
			StatusText:  cached.codeText,
			ContentType: cached.contentType,
		}
		markdown = cached.content
		ct = cached.contentType
		persistedPath = cached.persistedPath
		persistedSize = cached.persistedSize
	} else {
		u, err := parseAndUpgradeURL(urlInput)
		if err != nil {
			return nil, fmt.Errorf("webfetchtool: url: %w", err)
		}
		fetchURL := u.String()
		host := u.Hostname()

		if !skipWebFetchPreflight(rc) {
			dClient := (*http.Client)(nil)
			if rc != nil {
				dClient = rc.DomainCheckClient
			}
			if err := CheckDomainBlocklist(ctx, domainCheckOrigin(rc), host, dClient); err != nil {
				return nil, err
			}
		}

		raw, redir, fetchErr = getWithPermittedRedirects(ctx, client, fetchURL, 0)
		if redir != nil {
			st := redirectCodeText(redir.StatusCode)
			msg := fmt.Sprintf(`REDIRECT DETECTED: The URL redirects to a different host.

Original URL: %s
Redirect URL: %s
Status: %d %s

To complete your request, I need to fetch content from the redirected URL. Please use WebFetch again with these parameters:
- url: "%s"
- prompt: "%s"`, redir.OriginalURL, redir.RedirectURL, redir.StatusCode, st, redir.RedirectURL, prompt)
			out, _ := json.Marshal(webFetchOutput{
				Bytes:      len([]byte(msg)),
				Code:       redir.StatusCode,
				CodeText:   st,
				Result:     msg,
				DurationMs: time.Since(start).Milliseconds(),
				URL:        urlInput,
			})
			return out, nil
		}
		if fetchErr != nil {
			return nil, fetchErr
		}

		body := raw.Body
		rawBytes = len(body)
		ct = raw.ContentType

		if isBinaryContentType(ct) {
			p, sz, err := persistBinaryWebFetch(toolResultsDir(rc), body, ct)
			if err == nil {
				persistedPath, persistedSize = p, sz
			}
		}

		utf8Content := bytesToUTF8(body)
		if contentTypeIncludes(ct, "text/html") {
			md, err := htmlToMarkdown(utf8Content)
			if err != nil {
				markdown = htmlToPlainText(utf8Content)
			} else {
				markdown = md
			}
		} else {
			markdown = utf8Content
		}

		urlCacheSet(urlInput, urlCachePayload{
			bytes:         rawBytes,
			code:          raw.StatusCode,
			codeText:      raw.StatusText,
			content:       markdown,
			contentType:   ct,
			persistedPath: persistedPath,
			persistedSize: persistedSize,
		})
	}

	preapproved := IsPreapprovedURL(urlInput)
	nonInteractive := false
	if rc != nil && rc.NonInteractive != nil {
		nonInteractive = *rc.NonInteractive
	}
	var result string
	if preapproved && contentTypeIncludes(ct, "text/markdown") && len(markdown) < maxMarkdownLength {
		result = markdown
	} else {
		truncated := markdown
		if len(truncated) > maxMarkdownLength {
			truncated = truncateRunes(truncated, maxMarkdownLength) + "\n\n[Content truncated due to length...]"
		}
		if rc != nil && rc.ApplyPrompt != nil {
			var aerr error
			result, aerr = rc.ApplyPrompt(ctx, truncated, prompt, preapproved, nonInteractive)
			if aerr != nil {
				return nil, aerr
			}
		} else {
			result = MakeSecondaryModelPrompt(truncated, prompt, preapproved)
		}
	}

	if persistedPath != "" {
		sz := persistedSize
		if sz <= 0 {
			sz = rawBytes
		}
		result += fmt.Sprintf("\n\n[Binary content (%s, %s) also saved to %s]", ct, filereadtool.FormatFileSize(int64(sz)), persistedPath)
	}

	if len(result) > maxResultSizeChars {
		result = truncateRunes(result, maxResultSizeChars) + "\n\n[Result truncated to maxResultSizeChars]"
	}

	out, err := json.Marshal(webFetchOutput{
		Bytes:      rawBytes,
		Code:       raw.StatusCode,
		CodeText:   raw.StatusText,
		Result:     result,
		DurationMs: time.Since(start).Milliseconds(),
		URL:        urlInput,
	})
	return out, err
}
