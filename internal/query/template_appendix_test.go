package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTemplateMarkdownAppendix(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "foo.md"), []byte("# Hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadTemplateMarkdownAppendix(dir, []string{"foo"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "## Template foo") || !strings.Contains(s, "# Hi") {
		t.Fatalf("%q", s)
	}
}

func TestLoadTemplateMarkdownAppendix_rejectsPathInName(t *testing.T) {
	_, err := LoadTemplateMarkdownAppendix(t.TempDir(), []string{"../x"})
	if err == nil {
		t.Fatal("expected error")
	}
}
