package memdir

// Corresponds to restored-src/src/memdir/memoryTypes.ts (taxonomy + embedded prompt fragments).

import (
	"strings"

	_ "embed"
)

// MemoryTypes is memoryTypes.ts MEMORY_TYPES (closed taxonomy).
var MemoryTypes = []string{"user", "feedback", "project", "reference"}

// ParseMemoryType returns the type string if raw matches a known type; otherwise "" (parseMemoryType).
func ParseMemoryType(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	for _, t := range MemoryTypes {
		if s == t {
			return t
		}
	}
	return ""
}

//go:embed promptdata/types_combined.txt
var rawTypesSectionCombined string

//go:embed promptdata/types_individual.txt
var rawTypesSectionIndividual string

//go:embed promptdata/what_not_to_save.txt
var rawWhatNotToSave string

//go:embed promptdata/when_to_access.txt
var rawWhenToAccess string

//go:embed promptdata/trusting_recall.txt
var rawTrustingRecall string

//go:embed promptdata/frontmatter_example.txt
var rawFrontmatterExample string

//go:embed promptdata/when_to_access_combined.txt
var rawWhenToAccessCombined string

//go:embed promptdata/memory_persistence.txt
var rawMemoryPersistence string
