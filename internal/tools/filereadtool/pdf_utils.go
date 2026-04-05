package filereadtool

import (
	"path/filepath"
	"strconv"
	"strings"
)

// ParsePDFPageRange mirrors utils/pdfUtils.ts parsePDFPageRange.
func ParsePDFPageRange(pages string) (firstPage int, lastPage int, ok bool) {
	trimmed := strings.TrimSpace(pages)
	if trimmed == "" {
		return 0, 0, false
	}
	if strings.HasSuffix(trimmed, "-") {
		first, err := strconv.Atoi(strings.TrimSuffix(trimmed, "-"))
		if err != nil || first < 1 {
			return 0, 0, false
		}
		return first, -1, true // -1 sentinel = Infinity
	}
	if i := strings.IndexByte(trimmed, '-'); i >= 0 {
		first, err1 := strconv.Atoi(strings.TrimSpace(trimmed[:i]))
		last, err2 := strconv.Atoi(strings.TrimSpace(trimmed[i+1:]))
		if err1 != nil || err2 != nil || first < 1 || last < 1 || last < first {
			return 0, 0, false
		}
		return first, last, true
	}
	p, err := strconv.Atoi(trimmed)
	if err != nil || p < 1 {
		return 0, 0, false
	}
	return p, p, true
}

// PDFPageRangeWidth returns inclusive page count; lastPage -1 means unbounded (caller must clamp).
func PDFPageRangeWidth(first, last int) int {
	if last < 0 {
		return PDFMaxPagesPerRead + 1
	}
	return last - first + 1
}

// IsPDFExtension mirrors utils/pdfUtils.ts isPDFExtension.
func IsPDFExtension(ext string) bool {
	e := ext
	if strings.HasPrefix(e, ".") {
		e = e[1:]
	}
	return strings.ToLower(e) == "pdf"
}

// IsPDFSupported mirrors utils/pdfUtils.ts isPDFSupported (Haiku 3 predates PDF document blocks).
func IsPDFSupported(mainLoopModel string) bool {
	return !strings.Contains(strings.ToLower(mainLoopModel), "claude-3-haiku")
}

// ExtFromPath returns extension without dot, lowercased.
func ExtFromPath(p string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
}
