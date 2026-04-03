package memdir

import "strings"

// Valid memory frontmatter types (memoryTypes.ts MEMORY_TYPES).
var memoryTypes = []string{"user", "feedback", "project", "reference"}

// ParseMemoryType returns the type string if raw matches a known type; otherwise "".
func ParseMemoryType(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	for _, t := range memoryTypes {
		if s == t {
			return t
		}
	}
	return ""
}
