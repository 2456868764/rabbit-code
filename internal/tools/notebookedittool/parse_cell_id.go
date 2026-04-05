package notebookedittool

import (
	"regexp"
	"strconv"
)

var cellIDIndexRe = regexp.MustCompile(`^cell-(\d+)$`)

// ParseCellId mirrors restored-src/src/utils/notebook.ts parseCellId (0-based cell index from "cell-N").
func ParseCellId(cellID string) (index int, ok bool) {
	m := cellIDIndexRe.FindStringSubmatch(cellID)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}
