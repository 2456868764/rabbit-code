// TS parity helpers for normalizeAttachmentForAPI (memoryHeader, diagnostics, file line numbers, compact numbers).
package messages

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxDiagnosticsSummaryChars = 4000

// cyberRiskMitigationReminder matches FileReadTool.CYBER_RISK_MITIGATION_REMINDER (TS).
const cyberRiskMitigationReminder = `

<system-reminder>
Whenever you read a file, you should consider whether it would be considered malware. You CAN and SHOULD provide analysis of malware, what it is doing. But you MUST refuse to improve or augment the code. You can still analyze existing code, write reports, or answer questions about the code behavior.
</system-reminder>
`

func memoryAgeDays(mtimeMs int64) int {
	if mtimeMs <= 0 {
		return 0
	}
	d := int(math.Floor(float64(time.Now().UnixMilli()-mtimeMs) / 86400000.0))
	if d < 0 {
		return 0
	}
	return d
}

func memoryAge(mtimeMs int64) string {
	d := memoryAgeDays(mtimeMs)
	switch d {
	case 0:
		return "today"
	case 1:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", d)
	}
}

func memoryFreshnessText(mtimeMs int64) string {
	d := memoryAgeDays(mtimeMs)
	if d <= 1 {
		return ""
	}
	return fmt.Sprintf(
		"This memory is %d days old. Memories are point-in-time observations, not live state — claims about code behavior or file:line citations may be outdated. Verify against current code before asserting as fact.",
		d,
	)
}

// MemoryHeader mirrors TS memoryHeader (attachments.ts).
func MemoryHeader(path string, mtimeMs int64) string {
	stale := memoryFreshnessText(mtimeMs)
	if stale != "" {
		return stale + "\n\nMemory: " + path + ":"
	}
	return fmt.Sprintf("Memory (saved %s): %s:", memoryAge(mtimeMs), path)
}

func memoryFreshnessNoteFromMtime(mtimeMs int64) string {
	t := memoryFreshnessText(mtimeMs)
	if t == "" {
		return ""
	}
	return "<system-reminder>" + t + "</system-reminder>\n"
}

func memoryFreshnessNoteFromFileMap(f map[string]any) string {
	ms := int64FromAny(f["memoryMtimeMs"])
	if ms == 0 {
		ms = int64FromAny(f["_memoryMtimeMs"])
	}
	if ms == 0 {
		return ""
	}
	return memoryFreshnessNoteFromMtime(ms)
}

func int64FromAny(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func diagnosticSeveritySymbol(sev string) string {
	switch sev {
	case "Error":
		return "✖"
	case "Warning":
		return "⚠"
	case "Info":
		return "ℹ"
	case "Hint":
		return "★"
	default:
		return "•"
	}
}

// FormatDiagnosticsSummary mirrors DiagnosticTrackingService.formatDiagnosticsSummary (TS).
func FormatDiagnosticsSummary(files []any) string {
	const trunc = "…[truncated]"
	var sb strings.Builder
	first := true
	for _, raw := range files {
		fm, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		uri, _ := fm["uri"].(string)
		name := uri
		if name != "" {
			name = filepath.Base(strings.ReplaceAll(uri, "\\", "/"))
		}
		if name == "" {
			name = uri
		}
		diagArr, _ := fm["diagnostics"].([]any)
		if len(diagArr) == 0 {
			continue
		}
		if !first {
			sb.WriteString("\n\n")
		}
		first = false
		sb.WriteString(name)
		sb.WriteString(":\n")
		for i, d := range diagArr {
			dm, ok := d.(map[string]any)
			if !ok {
				continue
			}
			if i > 0 {
				sb.WriteByte('\n')
			}
			sev, _ := dm["severity"].(string)
			msg, _ := dm["message"].(string)
			code, _ := dm["code"].(string)
			src, _ := dm["source"].(string)
			line, char := 1, 1
			if rng, ok := dm["range"].(map[string]any); ok {
				if st, ok := rng["start"].(map[string]any); ok {
					line = intFromAny(st["line"]) + 1
					char = intFromAny(st["character"]) + 1
				}
			}
			sb.WriteString("  ")
			sb.WriteString(diagnosticSeveritySymbol(sev))
			sb.WriteString(fmt.Sprintf(" [Line %d:%d] %s", line, char, msg))
			if code != "" {
				sb.WriteString(fmt.Sprintf(" [%s]", code))
			}
			if src != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", src))
			}
		}
	}
	out := sb.String()
	if len(out) > maxDiagnosticsSummaryChars {
		keep := maxDiagnosticsSummaryChars - len(trunc)
		if keep < 0 {
			keep = 0
		}
		if keep > len(out) {
			keep = len(out)
		}
		return out[:keep] + trunc
	}
	return out
}

// formatNumberCompact mirrors TS formatNumber (en-US compact, lowercased suffix).
func formatNumberCompact(n float64) string {
	neg := n < 0
	x := n
	if neg {
		x = -x
	}
	sign := ""
	if neg {
		sign = "-"
	}
	if x < 1000 {
		if x == math.Trunc(x) {
			return sign + strconv.FormatInt(int64(x), 10)
		}
		return strings.ToLower(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%s%g", sign, n), "0"), "."))
	}
	var div float64
	var suf string
	switch {
	case x < 1_000_000:
		div, suf = 1000, "k"
	case x < 1_000_000_000:
		div, suf = 1_000_000, "m"
	default:
		div, suf = 1_000_000_000, "b"
	}
	v := x / div
	// TS uses minimumFractionDigits 1 for n >= 1000
	txt := fmt.Sprintf("%.1f", v)
	txt = strings.TrimRight(strings.TrimRight(txt, "0"), ".")
	return sign + strings.ToLower(txt+suf)
}

var bashLeadingWhitespaceNewlines = regexp.MustCompile(`^(\s*\n)+`)

func bashToolStdoutNormalize(stdout string) string {
	if stdout == "" {
		return ""
	}
	s := bashLeadingWhitespaceNewlines.ReplaceAllString(stdout, "")
	return strings.TrimRight(s, "\n\t \r")
}

// addLineNumbersForFileRead mirrors TS addLineNumbers (file.ts); default compact tab when kill-switch not set.
func addLineNumbersForFileRead(content string, startLine int, compact bool) string {
	if content == "" {
		return ""
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	arrow := "\u2192"
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		n := i + startLine
		if compact {
			b.WriteString(strconv.Itoa(n))
			b.WriteByte('\t')
			b.WriteString(line)
			continue
		}
		numStr := strconv.Itoa(n)
		if len(numStr) >= 6 {
			b.WriteString(numStr)
			b.WriteString(arrow)
			b.WriteString(line)
		} else {
			fmt.Fprintf(&b, "%6s%s%s", numStr, arrow, line)
		}
	}
	return b.String()
}

func fileReadTextToolResultString(fc map[string]any) string {
	f, nested := fileNested(fc)
	if !nested {
		return fileReadTextBody(fc)
	}
	totalLines := intFromAny(f["totalLines"])
	startLine := intFromAny(f["startLine"])
	if startLine < 1 {
		startLine = 1
	}
	content, hasStr := f["content"].(string)
	if !hasStr || content == "" {
		if totalLines == 0 {
			return "<system-reminder>Warning: the file exists but the contents are empty.</system-reminder>"
		}
		return fmt.Sprintf("<system-reminder>Warning: the file exists but is shorter than the provided offset (%d). The file has %d lines.</system-reminder>", startLine, totalLines)
	}
	compact := os.Getenv("RABBIT_TENGU_COMPACT_LINE_PREFIX_KILL_SWITCH") != "1"
	body := addLineNumbersForFileRead(content, startLine, compact)
	if note := memoryFreshnessNoteFromFileMap(f); note != "" {
		body = note + body
	}
	if shouldIncludeFileReadMitigation() {
		body += cyberRiskMitigationReminder
	}
	return body
}

func attachmentNonNilFloat64(att map[string]any, key string) (float64, bool) {
	v, ok := att[key]
	if !ok || v == nil {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

func shouldIncludeFileReadMitigation() bool {
	m := strings.ToLower(os.Getenv("RABBIT_MAIN_LOOP_MODEL"))
	if m == "" {
		m = strings.ToLower(os.Getenv("ANTHROPIC_MODEL"))
	}
	if m == "" {
		return true
	}
	return !strings.Contains(m, "opus-4-6")
}
