package filewritetool

import "strings"

// normalizeNewlines mirrors read-side normalization used in readFileState comparisons.
func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// StructuredPatchFullReplace builds a single hunk like getPatchForDisplay for full-file replace (unified +/- lines).
func StructuredPatchFullReplace(oldText, newText string) []map[string]any {
	oldNorm := normalizeNewlines(oldText)
	newNorm := normalizeNewlines(newText)
	oldLines := strings.Split(oldNorm, "\n")
	newLines := strings.Split(newNorm, "\n")
	var lines []string
	for _, l := range oldLines {
		lines = append(lines, "-"+l)
	}
	for _, l := range newLines {
		lines = append(lines, "+"+l)
	}
	return []map[string]any{{
		"oldStart": 1,
		"oldLines": len(oldLines),
		"newStart": 1,
		"newLines": len(newLines),
		"lines":    lines,
	}}
}
