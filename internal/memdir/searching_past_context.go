package memdir

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// BuildSearchingPastContextSection returns the "## Searching past context" block (memdir.ts)
// when RABBIT_CODE_MEMORY_SEARCH_PAST_CONTEXT is truthy. useShellGrep selects shell grep form
// vs Grep tool form (embedded-search / REPL parity).
func BuildSearchingPastContextSection(autoMemDir, projectDir string, useShellGrep bool) []string {
	if !features.MemorySearchPastContextEnabled() {
		return nil
	}
	autoMemDir = strings.TrimSpace(autoMemDir)
	projectDir = filepath.Clean(strings.TrimSpace(projectDir))
	if autoMemDir == "" || projectDir == "" {
		return nil
	}
	projWithSep := projectDir + string(filepath.Separator)
	var memSearch, transcriptSearch string
	if useShellGrep {
		memSearch = fmt.Sprintf(`grep -rn "<search term>" %s --include="*.md"`, autoMemDir)
		transcriptSearch = fmt.Sprintf(`grep -rn "<search term>" %s --include="*.jsonl"`, projWithSep)
	} else {
		memSearch = fmt.Sprintf(`Grep with pattern="<search term>" path="%s" glob="*.md"`, autoMemDir)
		transcriptSearch = fmt.Sprintf(`Grep with pattern="<search term>" path="%s" glob="*.jsonl"`, projWithSep)
	}
	return []string{
		"## Searching past context",
		"",
		"When looking for past context:",
		"1. Search topic files in your memory directory:",
		"```",
		memSearch,
		"```",
		"2. Session transcript logs (last resort — large files, slow):",
		"```",
		transcriptSearch,
		"```",
		"Use narrow search terms (error messages, file paths, function names) rather than broad keywords.",
		"",
	}
}
