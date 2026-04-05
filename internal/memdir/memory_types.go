package memdir

// Corresponds to restored-src/src/memdir/memoryTypes.ts (taxonomy + embedded prompt fragments).

import (
	"strings"

	_ "embed"
)

// MemoryTypes is memoryTypes.ts MEMORY_TYPES (closed taxonomy).
var MemoryTypes = []string{"user", "feedback", "project", "reference"}

// ParseMemoryTypeFromAny mirrors memoryTypes.ts parseMemoryType(raw: unknown): non-string → unset; unknown value → unset.
func ParseMemoryTypeFromAny(raw interface{}) (typ string, ok bool) {
	s, isStr := raw.(string)
	if !isStr {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	for _, t := range MemoryTypes {
		if s == t {
			return t, true
		}
	}
	return "", false
}

// ParseMemoryType returns the type string if raw matches a known type; otherwise "" (parseMemoryType).
func ParseMemoryType(raw string) string {
	t, ok := ParseMemoryTypeFromAny(raw)
	if !ok {
		return ""
	}
	return t
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

// MemoryDriftCaveat matches memoryTypes.ts MEMORY_DRIFT_CAVEAT (recall-side drift bullet).
const MemoryDriftCaveat = `- Memory records can become stale over time. Use memory as context for what was true at a given point in time. Before answering the user or building assumptions based solely on information in memory records, verify that the memory is still correct and up-to-date by reading the current state of the files or resources. If a recalled memory conflicts with current information, trust what you observe now — and update or remove the stale memory rather than acting on it.`

func splitPromptDataLines(raw string) []string {
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "\n")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

// TypesSectionCombined returns memoryTypes.ts TYPES_SECTION_COMBINED lines.
func TypesSectionCombined() []string { return splitPromptDataLines(rawTypesSectionCombined) }

// TypesSectionIndividual returns memoryTypes.ts TYPES_SECTION_INDIVIDUAL lines.
func TypesSectionIndividual() []string { return splitPromptDataLines(rawTypesSectionIndividual) }

// WhatNotToSaveSection returns memoryTypes.ts WHAT_NOT_TO_SAVE_SECTION lines.
func WhatNotToSaveSection() []string { return splitPromptDataLines(rawWhatNotToSave) }

// WhenToAccessSection returns memoryTypes.ts WHEN_TO_ACCESS_SECTION lines (individual / auto-only prompts).
func WhenToAccessSection() []string { return splitPromptDataLines(rawWhenToAccess) }

// WhenToAccessCombinedSection returns the combined-mode when-to-access block (team prompt).
func WhenToAccessCombinedSection() []string { return splitPromptDataLines(rawWhenToAccessCombined) }

// TrustingRecallSection returns memoryTypes.ts TRUSTING_RECALL_SECTION lines.
func TrustingRecallSection() []string { return splitPromptDataLines(rawTrustingRecall) }

// MemoryFrontmatterExample returns memoryTypes.ts MEMORY_FRONTMATTER_EXAMPLE lines.
func MemoryFrontmatterExample() []string { return splitPromptDataLines(rawFrontmatterExample) }

// MemoryFrontmatterExampleBlock is the frontmatter example as a single string for inline prompt assembly.
func MemoryFrontmatterExampleBlock() string {
	return strings.TrimSpace(rawFrontmatterExample)
}

// MemoryAndPersistenceSection returns the "## Memory and other forms of persistence" block (promptdata/memory_persistence.txt).
func MemoryAndPersistenceSection() []string { return splitPromptDataLines(rawMemoryPersistence) }
