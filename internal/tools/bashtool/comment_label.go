package bashtool

import "strings"

// ExtractBashCommentLabel mirrors commentLabel.ts extractBashCommentLabel.
func ExtractBashCommentLabel(command string) string {
	nl := strings.IndexByte(command, '\n')
	first := command
	if nl >= 0 {
		first = command[:nl]
	}
	firstLine := strings.TrimSpace(first)
	if !strings.HasPrefix(firstLine, "#") || strings.HasPrefix(firstLine, "#!") {
		return ""
	}
	s := strings.TrimSpace(strings.TrimLeft(firstLine, "#"))
	if s == "" {
		return ""
	}
	return s
}
