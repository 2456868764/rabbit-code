package fileedittool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripTrailingWhitespace_trimsLineEndingsPreservesBreaks(t *testing.T) {
	got := StripTrailingWhitespace("a  \nb\t \r\nc")
	want := "a\nb\r\nc"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStripTrailingWhitespace_noTrailingBreak(t *testing.T) {
	if got := StripTrailingWhitespace("x  "); got != "x" {
		t.Fatalf("got %q", got)
	}
}

func TestDesanitizeMatchString_fnr(t *testing.T) {
	got, reps := DesanitizeMatchString("pre<fnr>post")
	if got != "pre<function_results>post" {
		t.Fatalf("got %q", got)
	}
	if len(reps) != 1 || reps[0].from != "<fnr>" {
		t.Fatalf("reps %#v", reps)
	}
}

func TestDesanitizeMatchString_humanAssistant(t *testing.T) {
	got, _ := DesanitizeMatchString("\n\nH:hi")
	if got != "\n\nHuman:hi" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeSingleFileEditInput_missingFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.txt")
	in := editInput{OldString: "a", NewString: "b  "}
	out := NormalizeSingleFileEditInput("nope.txt", missing, in, nil)
	if out.OldString != "a" || out.NewString != "b  " {
		t.Fatalf("%+v", out)
	}
}

func TestNormalizeSingleFileEditInput_stripsNewWhenMatch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := editInput{OldString: "keep", NewString: "x  "}
	out := NormalizeSingleFileEditInput("f.txt", p, in, nil)
	if out.OldString != "keep" || out.NewString != "x" {
		t.Fatalf("%+v", out)
	}
}

func TestNormalizeSingleFileEditInput_markdownNoStripNew(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "note.md")
	if err := os.WriteFile(p, []byte("# h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := editInput{OldString: "# h", NewString: "## x  "}
	out := NormalizeSingleFileEditInput("note.md", p, in, nil)
	if out.NewString != "## x  " {
		t.Fatalf("want trailing spaces kept for md, got %q", out.NewString)
	}
}

func TestNormalizeSingleFileEditInput_desanitizeWhenDiskHasExpanded(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "san.txt")
	disk := "before<function_results>after"
	if err := os.WriteFile(p, []byte(disk), 0o644); err != nil {
		t.Fatal(err)
	}
	in := editInput{OldString: "before<fnr>after", NewString: "before<fnr>x  "}
	out := NormalizeSingleFileEditInput("san.txt", p, in, nil)
	if out.OldString != "before<function_results>after" {
		t.Fatalf("old %q", out.OldString)
	}
	if out.NewString != "before<function_results>x" {
		t.Fatalf("new %q", out.NewString)
	}
}
