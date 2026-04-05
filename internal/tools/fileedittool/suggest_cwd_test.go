package fileedittool

import (
	"os"
	"path/filepath"
	"testing"
)

// Mirrors utils/file.ts example: cwd = …/currentRepo, requested = …/parent/foobar (missing repo segment).
func TestSuggestPathUnderCwd_droppedRepoSegment(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(repo, "foobar.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	wrong := filepath.Join(root, "foobar.txt")
	if _, err := os.Stat(wrong); err == nil {
		t.Fatal("wrong path must not exist")
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	got, ok := SuggestPathUnderCwd(wrong)
	if !ok {
		t.Fatal("expected suggestion")
	}
	wantResolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		wantResolved = target
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		gotResolved = got
	}
	if filepath.Clean(gotResolved) != filepath.Clean(wantResolved) {
		t.Fatalf("got %q want %q (resolved)", got, target)
	}
}

func TestSuggestPathUnderCwd_noSuggestionWhenAlreadyUnderCwd(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	sub := filepath.Join(repo, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(sub, "a.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := SuggestPathUnderCwd(abs)
	if ok {
		t.Fatal("path already under cwd; no suggestion")
	}
}
