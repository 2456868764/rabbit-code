package memdir

import _ "embed"

// Prompt sections sourced from restored-src/src/memdir/memoryTypes.ts (H8 parity).

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
