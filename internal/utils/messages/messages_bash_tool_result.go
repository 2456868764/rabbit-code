// BashTool.mapToolResultToToolResultBlockParam string content parity for attachments (directory, etc.).
package messages

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	bashPreviewMaxBytes             = 2000 // TS PREVIEW_SIZE_BYTES
	bashAssistantBlockingBudgetSec  = 15   // ASSISTANT_BLOCKING_BUDGET_MS / 1000
	persistedOutputTag              = "<persisted-output>"
	persistedOutputClosingTag       = "</persisted-output>"
	notebookCellOutputTruncateBytes = 10000 // TS notebook LARGE_OUTPUT_THRESHOLD (text+image heuristic)
)

// bashDataURIRe mirrors TS DATA_URI_RE / parseDataUri (trimmed input).
var bashDataURIRe = regexp.MustCompile(`(?is)^data:([^;]+);base64,(.+)$`)

func bashParseDataURI(s string) (mediaType, data string, ok bool) {
	s = strings.TrimSpace(s)
	m := bashDataURIRe.FindStringSubmatch(s)
	if len(m) != 3 {
		return "", "", false
	}
	mt := strings.TrimSpace(m[1])
	data = strings.ReplaceAll(strings.TrimSpace(m[2]), "\n", "")
	data = strings.ReplaceAll(data, "\r", "")
	data = strings.ReplaceAll(data, "\t", "")
	data = strings.ReplaceAll(data, " ", "")
	return mt, data, true
}

func bashImageBlockFromStdout(stdout string) (map[string]any, bool) {
	mt, data, ok := bashParseDataURI(stdout)
	if !ok || data == "" {
		return nil, false
	}
	return map[string]any{
		"type": "image",
		"source": map[string]any{
			"type":       "base64",
			"media_type": mt,
			"data":       data,
		},
	}, true
}

func bashStructuredContentBlocks(sc []any) []map[string]any {
	var out []map[string]any
	for _, it := range sc {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, m)
	}
	return out
}

func bashBlocksHaveImage(blocks []map[string]any) bool {
	for _, b := range blocks {
		if mapStr(b, "type") == "image" {
			return true
		}
	}
	return false
}

func bashResolvedStdout(src, att map[string]any) string {
	stdout := strField(src, "stdout")
	if stdout == "" {
		stdout, _ = att["content"].(string)
	}
	if stdout == "" {
		stdout, _ = src["content"].(string)
	}
	return stdout
}

func bashGeneratePreview(content string, maxBytes int) (preview string, hasMore bool) {
	if len(content) <= maxBytes {
		return content, false
	}
	truncated := content[:maxBytes]
	lastNL := strings.LastIndexByte(truncated, '\n')
	cutPoint := maxBytes
	if lastNL > maxBytes/2 {
		cutPoint = lastNL
	}
	return content[:cutPoint], true
}

func bashBuildLargePersistedMessage(path string, originalSize int, preview string, hasMore bool) string {
	var b strings.Builder
	b.WriteString(persistedOutputTag)
	b.WriteByte('\n')
	fmt.Fprintf(&b, "Output too large (%s). Full output saved to: %s\n\n", formatFileSizeForAPI(originalSize), path)
	fmt.Fprintf(&b, "Preview (first %s):\n", formatFileSizeForAPI(bashPreviewMaxBytes))
	b.WriteString(preview)
	if hasMore {
		b.WriteString("\n...\n")
	} else {
		b.WriteByte('\n')
	}
	b.WriteString(persistedOutputClosingTag)
	return b.String()
}

// bashAttachmentSource returns the map holding bash fields (top-level or bash/bashResult envelope).
func bashAttachmentSource(att map[string]any) map[string]any {
	if br, ok := att["bash"].(map[string]any); ok && len(br) > 0 {
		return br
	}
	if br, ok := att["bashResult"].(map[string]any); ok && len(br) > 0 {
		return br
	}
	return att
}

// bashPlaintextToolResultBody mirrors BashTool.mapToolResultToToolResultBlockParam string-join segment
// (stdout normalize, persisted preview, stderr, background path) — no structured / isImage branches.
func bashPlaintextToolResultBody(src, att map[string]any) string {
	stdout := bashResolvedStdout(src, att)
	processed := bashToolStdoutNormalize(stdout)

	if path := strings.TrimSpace(strField(src, "persistedOutputPath")); path != "" {
		orig := intFromAny(src["persistedOutputSize"])
		if orig <= 0 && processed != "" {
			orig = len(processed)
		}
		prev, more := bashGeneratePreview(processed, bashPreviewMaxBytes)
		processed = bashBuildLargePersistedMessage(path, orig, prev, more)
	}

	errMsg := strings.TrimSpace(strField(src, "stderr"))
	if truthy(src["interrupted"]) {
		if errMsg != "" {
			errMsg += "\n"
		}
		errMsg += "<error>Command was aborted before completion</error>"
	}

	bgID := strField(src, "backgroundTaskId")
	var bgInfo string
	if bgID != "" {
		outPath := strField(src, "backgroundTaskOutputPath")
		if outPath == "" {
			outPath = strField(src, "taskOutputPath")
		}
		switch {
		case truthy(src["assistantAutoBackgrounded"]):
			bgInfo = fmt.Sprintf(
				"Command exceeded the assistant-mode blocking budget (%ds) and was moved to the background with ID: %s. It is still running — you will be notified when it completes. Output is being written to: %s. In assistant mode, delegate long-running work to a subagent or use run_in_background to keep this conversation responsive.",
				bashAssistantBlockingBudgetSec, bgID, outPath,
			)
		case truthy(src["backgroundedByUser"]):
			bgInfo = fmt.Sprintf(
				"Command was manually backgrounded by user with ID: %s. Output is being written to: %s",
				bgID, outPath,
			)
		default:
			bgInfo = fmt.Sprintf(
				"Command running in background with ID: %s. Output is being written to: %s",
				bgID, outPath,
			)
		}
	}

	var parts []string
	if processed != "" {
		parts = append(parts, processed)
	}
	if errMsg != "" {
		parts = append(parts, errMsg)
	}
	if bgInfo != "" {
		parts = append(parts, bgInfo)
	}
	return strings.Join(parts, "\n")
}

// BashAttachmentToolResultContentString mirrors BashTool.mapToolResultToToolResultBlockParam content string
// (before createToolResultMessage wrapper). Extended transcript fields are optional; default uses content as stdout.
func BashAttachmentToolResultContentString(att map[string]any) string {
	src := bashAttachmentSource(att)

	if sc, ok := src["structuredContent"].([]any); ok && len(sc) > 0 {
		s := strings.TrimSpace(tsJSONString(sc))
		if s != "" {
			return s
		}
	}

	stdout := bashResolvedStdout(src, att)
	if truthy(src["isImage"]) && strings.TrimSpace(stdout) != "" {
		if _, _, ok := bashParseDataURI(stdout); ok {
			return "[Image output from Bash — omitted in attachment string preview; full API uses image content blocks]"
		}
	}

	return bashPlaintextToolResultBody(src, att)
}

// BashToolResultMetaMessage mirrors TS createToolResultMessage(BashTool, toolUseResult) for attachment expansion:
// image blocks pass through as user message content; otherwise "Result of calling …" + string or JSON.
func BashToolResultMetaMessage(toolName string, att map[string]any) (msg TSMsg) {
	defer func() {
		if recover() != nil {
			msg = CreateUserMessage(CreateUserMessageOpts{
				Content: fmt.Sprintf("Result of calling the %s tool: Error", toolName),
				IsMeta:  true,
			})
		}
	}()

	src := bashAttachmentSource(att)

	if sc, ok := src["structuredContent"].([]any); ok && len(sc) > 0 {
		blocks := bashStructuredContentBlocks(sc)
		if bashBlocksHaveImage(blocks) {
			return CreateUserMessage(CreateUserMessageOpts{
				Content: blocksToAnySlice(blocks),
				IsMeta:  true,
			})
		}
		s := strings.TrimSpace(tsJSONString(sc))
		if s != "" {
			return CreateUserMessage(CreateUserMessageOpts{
				Content: fmt.Sprintf("Result of calling the %s tool:\n%s", toolName, s),
				IsMeta:  true,
			})
		}
	}

	stdout := bashResolvedStdout(src, att)
	if truthy(src["isImage"]) && strings.TrimSpace(stdout) != "" {
		if im, ok := bashImageBlockFromStdout(stdout); ok {
			return CreateUserMessage(CreateUserMessageOpts{
				Content: []any{im},
				IsMeta:  true,
			})
		}
	}

	body := bashPlaintextToolResultBody(src, att)
	return CreateUserMessage(CreateUserMessageOpts{
		Content: fmt.Sprintf("Result of calling the %s tool:\n%s", toolName, body),
		IsMeta:  true,
	})
}
