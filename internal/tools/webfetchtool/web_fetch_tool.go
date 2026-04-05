package webfetchtool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

// Run fetches a URL and produces result text aligned with WebFetchTool.call (no domain preflight; optional ApplyPrompt in RunContext).
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

	u, err := parseAndUpgradeURL(urlInput)
	if err != nil {
		return nil, fmt.Errorf("webfetchtool: url: %w", err)
	}
	fetchURL := u.String()

	rc := RunContextFrom(ctx)
	client := defaultFetchClient()
	if rc != nil && rc.HTTPClient != nil {
		client = rc.HTTPClient
	}

	raw, redir, err := getWithPermittedRedirects(ctx, client, fetchURL, 0)
	if redir != nil {
		st := httpStatusText(redir.StatusCode, "")
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
	if err != nil {
		return nil, err
	}

	body := raw.Body
	rawBytes := len(body)
	ct := raw.ContentType
	utf8Content := bytesToUTF8(body)

	var markdown string
	if isBinaryContentType(ct) {
		mt := strings.TrimSpace(strings.Split(ct, ";")[0])
		markdown = fmt.Sprintf("[Binary content (%s, %d bytes) — Rabbit Code WebFetch does not persist binary tool results to disk; use Read or an MCP tool if you need the file.]", mt, rawBytes)
	} else if strings.Contains(strings.ToLower(ct), "text/html") {
		markdown = htmlToPlainText(utf8Content)
	} else {
		markdown = utf8Content
	}

	preapproved := IsPreapprovedURL(urlInput)
	ctLower := strings.ToLower(ct)
	var result string
	if preapproved && strings.Contains(ctLower, "text/markdown") && len(markdown) < maxMarkdownLength {
		result = markdown
	} else {
		truncated := markdown
		if len(truncated) > maxMarkdownLength {
			truncated = truncateRunes(truncated, maxMarkdownLength) + "\n\n[Content truncated due to length...]"
		}
		if rc != nil && rc.ApplyPrompt != nil {
			var aerr error
			result, aerr = rc.ApplyPrompt(ctx, truncated, prompt, preapproved)
			if aerr != nil {
				return nil, aerr
			}
		} else {
			result = MakeSecondaryModelPrompt(truncated, prompt, preapproved)
		}
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
