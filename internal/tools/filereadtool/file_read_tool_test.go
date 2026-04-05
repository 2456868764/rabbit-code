package filereadtool_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

func TestFileRead_Run_success(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("line1\nline2\nline3"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fr := filereadtool.New()
	out, err := fr.Run(context.Background(), []byte(`{"file_path":`+jsonQuote(abs)+`}`))
	if err != nil {
		t.Fatal(err)
	}
	var wrap struct {
		Type string `json:"type"`
		File struct {
			Content    string `json:"content"`
			NumLines   int    `json:"numLines"`
			StartLine  int    `json:"startLine"`
			TotalLines int    `json:"totalLines"`
		} `json:"file"`
	}
	if err := json.Unmarshal(out, &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.Type != "text" || wrap.File.TotalLines != 3 || wrap.File.NumLines != 3 {
		t.Fatalf("%+v %s", wrap, out)
	}
	if wrap.File.Content != "line1\nline2\nline3" {
		t.Fatalf("content %q", wrap.File.Content)
	}
}

func TestFileRead_Run_badInput(t *testing.T) {
	fr := filereadtool.New()
	_, err := fr.Run(context.Background(), []byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = fr.Run(context.Background(), []byte(`{"file_path":""}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileRead_Run_rejectBinaryExtension(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.exe")
	if err := os.WriteFile(p, []byte("MZfake"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fr := filereadtool.New()
	_, err := fr.Run(context.Background(), []byte(`{"file_path":`+jsonQuote(abs)+`}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileRead_Run_blockedDevice(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/zero")
	}
	fr := filereadtool.New()
	_, err := fr.Run(context.Background(), []byte(`{"file_path":"/dev/zero"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileRead_Run_offsetBeyond(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "one.txt")
	if err := os.WriteFile(p, []byte("only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fr := filereadtool.New()
	_, err := fr.Run(context.Background(), []byte(`{"file_path":`+jsonQuote(abs)+`,"offset":3}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileRead_implementsToolsTool(t *testing.T) {
	var _ tools.Tool = filereadtool.New()
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
