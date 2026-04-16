package bashtool

import (
	"regexp"
	"strconv"
	"strings"
)

// StripEmptyLines mirrors BashTool/utils.ts stripEmptyLines.
func StripEmptyLines(content string) string {
	lines := strings.Split(content, "\n")
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines) - 1
	for end >= 0 && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if start > end {
		return ""
	}
	return strings.Join(lines[start:end+1], "\n")
}

var imageDataURIRE = regexp.MustCompile(`(?i)^data:image/[a-z0-9.+_-]+;base64,`)
var dataURIRE = regexp.MustCompile(`^data:([^;]+);base64,(.+)$`)

// IsImageOutput mirrors utils.ts isImageOutput.
func IsImageOutput(content string) bool {
	return imageDataURIRE.MatchString(content)
}

// DataURIResult holds the result of ParseDataUri.
type DataURIResult struct {
	MediaType string
	Data      string
}

// ParseDataUri mirrors utils.ts parseDataUri.
func ParseDataUri(s string) *DataURIResult {
	m := dataURIRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil || m[1] == "" || m[2] == "" {
		return nil
	}
	return &DataURIResult{MediaType: m[1], Data: m[2]}
}

// FormatOutputResult holds the result of FormatOutput.
type FormatOutputResult struct {
	TotalLines       int
	TruncatedContent string
	IsImage          bool
}

// FormatOutput mirrors utils.ts formatOutput.
// maxOutputLength < 0 means no truncation (headless callers that handle output limits elsewhere).
func FormatOutput(content string, maxOutputLength int) FormatOutputResult {
	if IsImageOutput(content) {
		return FormatOutputResult{TotalLines: 1, TruncatedContent: content, IsImage: true}
	}
	totalLines := strings.Count(content, "\n") + 1
	if maxOutputLength < 0 || len(content) <= maxOutputLength {
		return FormatOutputResult{TotalLines: totalLines, TruncatedContent: content}
	}
	truncated := content[:maxOutputLength]
	remainingLines := strings.Count(content[maxOutputLength:], "\n") + 1
	return FormatOutputResult{
		TotalLines:       totalLines,
		TruncatedContent: truncated + "\n\n... [" + strconv.Itoa(remainingLines) + " lines truncated] ...",
	}
}

// StdErrAppendShellResetMessage mirrors utils.ts stdErrAppendShellResetMessage.
// originalCwd is the value of getOriginalCwd() at call time (caller must supply it for headless).
func StdErrAppendShellResetMessage(stderr, originalCwd string) string {
	return strings.TrimRight(stderr, " \t\n\r") + "\nShell cwd was reset to " + originalCwd
}

// ContentBlock is a minimal headless analogue of @anthropic-ai/sdk ContentBlockParam for CreateContentSummary.
type ContentBlock struct {
	Type string
	Text string
}

func pluralWord(n int, singular string) string {
	if n == 1 {
		return singular
	}
	return singular + "s"
}

// CreateContentSummary mirrors utils.ts createContentSummary.
func CreateContentSummary(blocks []ContentBlock) string {
	var parts []string
	textCount, imageCount := 0, 0
	for _, b := range blocks {
		if b.Type == "image" {
			imageCount++
		} else if b.Type == "text" && b.Text != "" {
			textCount++
			preview := b.Text
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			parts = append(parts, preview)
		}
	}
	var summary []string
	if imageCount > 0 {
		summary = append(summary, "["+strconv.Itoa(imageCount)+" "+pluralWord(imageCount, "image")+"]")
	}
	if textCount > 0 {
		summary = append(summary, "["+strconv.Itoa(textCount)+" text "+pluralWord(textCount, "block")+"]")
	}
	result := "MCP Result: " + strings.Join(summary, ", ")
	if len(parts) > 0 {
		result += "\n\n" + strings.Join(parts, "\n\n")
	}
	return result
}
