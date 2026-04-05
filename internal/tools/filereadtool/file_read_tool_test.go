package filereadtool_test

import (
	"context"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestFileRead_Run_strictJSONUnknownField(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(p)
	fr := filereadtool.New()
	_, err := fr.Run(context.Background(), []byte(`{"file_path":`+jsonQuote(abs)+`,"extra":1}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("got %v", err)
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
	out, err := fr.Run(context.Background(), []byte(`{"file_path":`+jsonQuote(abs)+`,"offset":3}`))
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
	if wrap.Type != "text" || wrap.File.TotalLines != 2 || wrap.File.NumLines != 0 || wrap.File.Content != "" {
		t.Fatalf("unexpected %+v / %s", wrap, out)
	}
}

func TestFileRead_implementsToolsTool(t *testing.T) {
	var _ tools.Tool = filereadtool.New()
}

func TestFileRead_Run_notebook(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "n.ipynb")
	nb := `{"cells":[{"cell_type":"markdown","source":["# t"],"metadata":{}}],"metadata":{}}`
	if err := os.WriteFile(p, []byte(nb), 0o644); err != nil {
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
			FilePath string `json:"filePath"`
			Cells    []any  `json:"cells"`
		} `json:"file"`
	}
	if err := json.Unmarshal(out, &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.Type != "notebook" || len(wrap.File.Cells) != 1 {
		t.Fatalf("%+v %s", wrap, out)
	}
}

func TestFileRead_Run_png(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
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
			Base64 string `json:"base64"`
			Type   string `json:"type"`
		} `json:"file"`
	}
	if err := json.Unmarshal(out, &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.Type != "image" || wrap.File.Base64 == "" || wrap.File.Type == "" {
		t.Fatalf("%+v %s", wrap, out)
	}
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
