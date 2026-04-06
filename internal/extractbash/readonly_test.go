package extractbash

import (
	"encoding/json"
	"testing"
)

func TestIsReadOnlyBashInputJSON(t *testing.T) {
	if !IsReadOnlyBashInputJSON([]byte(`{"cmd":"ls -la"}`)) {
		t.Fatal("ls")
	}
	if IsReadOnlyBashInputJSON([]byte(`{"cmd":"rm -rf /"}`)) {
		t.Fatal("deny rm")
	}
	if IsReadOnlyBashInputJSON([]byte(`{"cmd":"ls | wc"}`)) {
		t.Fatal("deny pipe")
	}
	if !IsReadOnlyBashInputJSON([]byte(`{"cmd":"git log -1 --oneline"}`)) {
		t.Fatal("allow read-only git")
	}
	if IsReadOnlyBashInputJSON([]byte(`{"cmd":"git push"}`)) {
		t.Fatal("deny git push")
	}
	if !IsReadOnlyBashInputJSON([]byte(`{"cmd":"git blame README.md"}`)) {
		t.Fatal("allow git blame")
	}
	if !IsReadOnlyBashInputJSON([]byte(`{"cmd":"git stash list"}`)) {
		t.Fatal("allow git stash list")
	}
	if !IsReadOnlyBashInputJSON([]byte(`{"cmd":"git remote -v"}`)) {
		t.Fatal("allow git remote -v")
	}
	if !IsReadOnlyBashInputJSON([]byte(`{"cmd":"git remote show origin"}`)) {
		t.Fatal("allow git remote show")
	}
	if IsReadOnlyBashInputJSON([]byte(`{"cmd":"git remote add origin u"}`)) {
		t.Fatal("deny git remote add")
	}
	if !IsReadOnlyBashInputJSON([]byte(`{"cmd":"git config --get core.editor"}`)) {
		t.Fatal("allow git config --get")
	}
	if IsReadOnlyBashInputJSON([]byte(`{"cmd":"git config --set x y"}`)) {
		t.Fatal("deny git config set-style")
	}
	in, _ := json.Marshal(map[string]string{"cmd": "\x00"})
	if IsReadOnlyBashInputJSON(in) {
		t.Fatal("deny null in cmd")
	}
}
