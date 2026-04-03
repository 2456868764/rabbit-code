package compact

import "testing"

func TestIsCompactableToolName(t *testing.T) {
	if !IsCompactableToolName("Read") || !IsCompactableToolName("Bash") || !IsCompactableToolName("PowerShell") {
		t.Fatal("expected core tools compactable")
	}
	if !IsCompactableToolName("Grep") || !IsCompactableToolName("Glob") {
		t.Fatal("expected search tools compactable")
	}
	if !IsCompactableToolName("WebSearch") || !IsCompactableToolName("WebFetch") {
		t.Fatal("expected web tools compactable")
	}
	if !IsCompactableToolName("Edit") || !IsCompactableToolName("Write") {
		t.Fatal("expected file mutation tools compactable")
	}
	if IsCompactableToolName("NotebookEdit") || IsCompactableToolName("") {
		t.Fatal("expected non-compactable")
	}
}
