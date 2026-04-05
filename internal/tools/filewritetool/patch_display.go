package filewritetool

import (
	"sort"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const patchDisplayContextLines = 3

// patchDiffTimeout mirrors diff.ts DIFF_TIMEOUT_MS (5s).
const patchDiffTimeout = 5 * time.Second

const ampersandToken = "<<:AMPERSAND_TOKEN:>>"
const dollarToken = "<<:DOLLAR_TOKEN:>>"

func escapeForDiff(s string) string {
	s = strings.ReplaceAll(s, "&", ampersandToken)
	return strings.ReplaceAll(s, "$", dollarToken)
}

func unescapeFromDiff(s string) string {
	s = strings.ReplaceAll(s, ampersandToken, "&")
	return strings.ReplaceAll(s, dollarToken, "$")
}

// convertLeadingTabsToSpaces mirrors utils/file.ts convertLeadingTabsToSpaces (leading \t → two spaces per tab).
func convertLeadingTabsToSpaces(content string) string {
	if !strings.Contains(content, "\t") {
		return content
	}
	var b strings.Builder
	rest := content
	for rest != "" {
		nl := strings.IndexByte(rest, '\n')
		var line string
		if nl < 0 {
			line = rest
			rest = ""
		} else {
			line = rest[:nl+1]
			rest = rest[nl+1:]
		}
		body := line
		suffix := ""
		if strings.HasSuffix(body, "\n") {
			body = body[:len(body)-1]
			suffix = "\n"
		}
		if strings.HasSuffix(body, "\r") {
			body = body[:len(body)-1]
			suffix = "\r" + suffix
		}
		i := 0
		for i < len(body) && body[i] == '\t' {
			i++
		}
		if i > 0 {
			b.WriteString(strings.Repeat("  ", i))
			b.WriteString(body[i:])
		} else {
			b.WriteString(body)
		}
		b.WriteString(suffix)
	}
	return b.String()
}

type lineRec struct {
	op   diffmatchpatch.Operation
	text string
}

func expandDiffsToLineRecs(diffs []diffmatchpatch.Diff) []lineRec {
	var out []lineRec
	for _, d := range diffs {
		if d.Text == "" {
			continue
		}
		s := d.Text
		for len(s) > 0 {
			idx := strings.IndexByte(s, '\n')
			if idx < 0 {
				out = append(out, lineRec{op: d.Type, text: s})
				break
			}
			out = append(out, lineRec{op: d.Type, text: s[:idx+1]})
			s = s[idx+1:]
		}
	}
	return out
}

func lineNumsBefore(recs []lineRec, end int) (oldStart, newStart int) {
	oldStart, newStart = 1, 1
	for i := 0; i < end && i < len(recs); i++ {
		switch recs[i].op {
		case diffmatchpatch.DiffEqual:
			oldStart++
			newStart++
		case diffmatchpatch.DiffDelete:
			oldStart++
		case diffmatchpatch.DiffInsert:
			newStart++
		}
	}
	return oldStart, newStart
}

func hunkBodyLine(op diffmatchpatch.Operation, rawLine string) string {
	body := strings.TrimSuffix(rawLine, "\n")
	body = strings.TrimSuffix(body, "\r")
	switch op {
	case diffmatchpatch.DiffEqual:
		return " " + unescapeFromDiff(body)
	case diffmatchpatch.DiffDelete:
		return "-" + unescapeFromDiff(body)
	case diffmatchpatch.DiffInsert:
		return "+" + unescapeFromDiff(body)
	default:
		return unescapeFromDiff(body)
	}
}

func mergeIntervals(iv [][2]int) [][2]int {
	if len(iv) == 0 {
		return nil
	}
	sort.Slice(iv, func(i, j int) bool {
		if iv[i][0] != iv[j][0] {
			return iv[i][0] < iv[j][0]
		}
		return iv[i][1] < iv[j][1]
	})
	out := [][2]int{iv[0]}
	for _, cur := range iv[1:] {
		last := &out[len(out)-1]
		if cur[0] <= last[1] {
			if cur[1] > last[1] {
				last[1] = cur[1]
			}
		} else {
			out = append(out, cur)
		}
	}
	return out
}

func dirtyIntervals(recs []lineRec, ctx int) [][2]int {
	n := len(recs)
	var raw [][2]int
	for i := range recs {
		if recs[i].op == diffmatchpatch.DiffEqual {
			continue
		}
		lo := i - ctx
		if lo < 0 {
			lo = 0
		}
		hi := i + ctx + 1
		if hi > n {
			hi = n
		}
		raw = append(raw, [2]int{lo, hi})
	}
	return mergeIntervals(raw)
}

// GetPatchForDisplay mirrors utils/diff.ts getPatchForDisplay for a single full-file replace (FileWriteTool call path).
// filePath is passed through for parity with the upstream API (embedded in the diff engine’s file identity).
func GetPatchForDisplay(filePath, oldContent, newContent string) []map[string]any {
	_ = filePath
	oldPrep := escapeForDiff(convertLeadingTabsToSpaces(oldContent))
	newPrep := escapeForDiff(convertLeadingTabsToSpaces(newContent))
	dmp := diffmatchpatch.New()
	dmp.DiffTimeout = patchDiffTimeout
	ch1, ch2, lineArray := dmp.DiffLinesToChars(oldPrep, newPrep)
	diffs := dmp.DiffMain(ch1, ch2, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)
	recs := expandDiffsToLineRecs(diffs)
	if len(recs) == 0 {
		return nil
	}
	intervals := dirtyIntervals(recs, patchDisplayContextLines)
	if len(intervals) == 0 {
		if oldContent == newContent {
			return nil
		}
		return StructuredPatchFullReplace(oldContent, newContent)
	}
	var hunks []map[string]any
	for _, iv := range intervals {
		lo, hi := iv[0], iv[1]
		oldStart, newStart := lineNumsBefore(recs, lo)
		var lines []string
		oldCnt, newCnt := 0, 0
		for i := lo; i < hi; i++ {
			r := recs[i]
			lines = append(lines, hunkBodyLine(r.op, r.text))
			switch r.op {
			case diffmatchpatch.DiffEqual:
				oldCnt++
				newCnt++
			case diffmatchpatch.DiffDelete:
				oldCnt++
			case diffmatchpatch.DiffInsert:
				newCnt++
			}
		}
		hunks = append(hunks, map[string]any{
			"oldStart": oldStart,
			"oldLines": oldCnt,
			"newStart": newStart,
			"newLines": newCnt,
			"lines":    lines,
		})
	}
	return hunks
}
