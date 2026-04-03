package compact

import "testing"

// Expected names match restored-src/src/services/compact/microCompact.ts COMPACTABLE_TOOLS
// (FILE_READ_TOOL_NAME, SHELL_TOOL_NAMES, GREP, GLOB, WEB_SEARCH, WEB_FETCH, FILE_EDIT, FILE_WRITE).
func TestCompactableToolNames_matchMicroCompactTS(t *testing.T) {
	want := []string{
		"Read", "Bash", "PowerShell", "Grep", "Glob",
		"WebSearch", "WebFetch", "Edit", "Write",
	}
	for _, name := range want {
		if !IsCompactableToolName(name) {
			t.Errorf("expected %q to be compactable (drift vs microCompact.ts COMPACTABLE_TOOLS?)", name)
		}
	}
	if IsCompactableToolName("TodoWrite") {
		t.Fatal("TodoWrite should not be compactable")
	}
}
