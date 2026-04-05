package stophooks

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestListMarkdownBasenames(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "z.md"), []byte("x"), 0o644)
	got, err := ListMarkdownBasenames(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a.md" || got[1] != "z.md" {
		t.Fatalf("%v", got)
	}
}

func TestRun_listJSON(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "hook.md"), []byte("#"), 0o644)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"list", "-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var m struct {
		Markdown []string `json:"markdown"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Markdown) != 1 || m.Markdown[0] != "hook.md" {
		t.Fatalf("%+v", m)
	}
}

func TestRun_listMissingDirEnv(t *testing.T) {
	t.Setenv("RABBIT_CODE_STOP_HOOKS_DIR", "")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"list"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
}
