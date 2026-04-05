package filewritetool

import "testing"

func TestGetPatchForDisplay_singleChangeWithContext(t *testing.T) {
	old := "l1\nl2\nl3\nl4\nl5\n"
	new := "l1\nl2\nchanged\nl4\nl5\n"
	h := GetPatchForDisplay("f.txt", old, new)
	if len(h) < 1 {
		t.Fatalf("expected hunks, got %v", h)
	}
	lines, _ := h[0]["lines"].([]string)
	if len(lines) < 3 {
		t.Fatalf("expected context lines, got %v", lines)
	}
}

func TestGetPatchForDisplay_identicalEmptyPatch(t *testing.T) {
	s := "same\n"
	h := GetPatchForDisplay("f.txt", s, s)
	if h != nil {
		t.Fatalf("expected nil, got %v", h)
	}
}
