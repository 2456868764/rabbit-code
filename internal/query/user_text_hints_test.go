package query

import "testing"

func TestApplyUserTextHints(t *testing.T) {
	out := ApplyUserTextHints("hello", UserTextHintFlags{Ultrathink: true})
	if out == "" || out == "hello" {
		t.Fatal(out)
	}
	out2 := ApplyUserTextHints("x", UserTextHintFlags{ContextCollapse: true, Ultraplan: true})
	if out2 == "" || out2 == "x" {
		t.Fatal(out2)
	}
	out3 := ApplyUserTextHints("z", UserTextHintFlags{SessionRestore: true})
	if out3 == "" || out3 == "z" {
		t.Fatal(out3)
	}
}

func TestFormatHeadlessModeTags_order(t *testing.T) {
	got := FormatHeadlessModeTags(UserTextHintFlags{
		Ultraplan: true, Ultrathink: true, ContextCollapse: true, SessionRestore: true,
	})
	want := "context_collapse,ultrathink,ultraplan,session_restore"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
